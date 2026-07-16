package ldap

import (
	"context"
	"strings"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/go-ldap/ldap/v3"
	"github.com/vjeantet/goldap/message"
	"github.com/vjeantet/ldapserver"
)

//	var RateLimit = rate.Limit[rate.ACL[*ldapserver.Message], rate.ACL[*ldapserver.Message], *ldapserver.Message]{
//		Domain:      []string{"ldap"},
//		MaxAttempts: 5,
//		Period:      time.Hour,
//		BanDuration: time.Hour * 24,
//		KeyGen: func(domain []string, m *ldapserver.Message) (string, error) {
//			// Limit based on the account being brute-forced, not the Docker IP
//			r := m.GetBindRequest()
//			domain = append(domain, string(r.Name()))
//			return strings.Join(domain, ":"), nil
//		},
//	}

// -------------------------------------------------------------
// BIND Handler (Authentication & Session Creation)
// -------------------------------------------------------------
func handleBind(w ldapserver.ResponseWriter, m *ldapserver.Message) {

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	//	// Enforce Rate Limiting using your custom package
	//	if err := RateLimit.Check(ctx, m); err != nil {
	//		_log.Printf("[BIND] Rate limit exceeded/blocked for IP: %s", m.Client.Addr().String())
	//		res := ldapserver.NewBindResponse(ldapserver.LDAPResultBusy)
	//		res.SetDiagnosticMessage("Too many attempts. Try again later.")
	//		w.Write(res)
	//		return
	//	}

	r := m.GetBindRequest()
	bindDN := string(r.Name())
	password := string(r.AuthenticationSimple())

	_log.Infof("[BIND] Attempt: DN=%s", bindDN)

	if len(password) > 64 || len(password) < 4 || strings.TrimSpace(password) == "" {
		_log.Warnf("[BIND] Rejected, DN=%s, PasswordLen=%d", bindDN, len(password))
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultUnwillingToPerform)
		res.SetDiagnosticMessage("Authentication not aligned with standards.")
		w.Write(res)
		return
	}

	parsedDN, err := ldap.ParseDN(bindDN)
	if err != nil || len(parsedDN.RDNs) == 0 {
		_log.Warnf("[BIND] Malformed DN rejected: %v", err)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), django.ConfigGet(
		django.Global.Settings,
		APPVAR_LDAP_TIMEOUT,
		DEFAULT_LDAP_TIMEOUT,
	))
	defer cancel()

	identityRDN := parsedDN.RDNs[0].Attributes[0]
	attrType := strings.ToLower(identityRDN.Type)
	attrValue := identityRDN.Value
	qs := auth.GetUserQuerySet().WithContext(ctx)

	switch attrType {
	case "cn", "uid":
		qs = qs.Filter("Username__iexact", attrValue)
	case "mail":
		qs = qs.Filter("Email__iexact", attrValue)
	default:
		_log.Warnf("[BIND] Unsupported bind attribute: %s", attrType)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	userRow, err := qs.Get()
	if err != nil {
		_log.Warnf("[BIND] Failed to retrieve %s=%s (%v)", attrType, attrValue, err)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	u := userRow.Object
	if !u.IsActive {
		_log.Warnf("[BIND] Access denied: User %s is inactive", u.Username)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	if err := u.Password.Check(password); err != nil {
		_log.Warnf("[BIND] Invalid password for user: %s: %q", u.Username, password)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	// Session established securely!
	// _ = RateLimit.Reset(ctx, m)
	m.Client.SetData(u.IsAdministrator)

	_log.Infof("[BIND] SUCCESS for user: %s", u.Username)
	res := ldapserver.NewBindResponse(ldapserver.LDAPResultSuccess)
	w.Write(res)
}

// -------------------------------------------------------------
// SEARCH Handler (Users & Aliases)
// -------------------------------------------------------------
func handleSearch(w ldapserver.ResponseWriter, m *ldapserver.Message) {
	sessionData := m.Client.GetData()
	isLdapAdmin, ok := sessionData.(bool)
	if !ok || !isLdapAdmin {
		_log.Warnf("[SEARCH] Rejected unauthorized search attempt")
		w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultInsufficientAccessRights))
		return
	}

	r := m.GetSearchRequest()
	baseDN := string(r.BaseObject())

	// 1. Flatten the AST into a map
	params := make(map[string]string)
	flattenFilterAST(r.Filter(), params)

	_log.Infof("[SEARCH] Base: %s | Params: %v", baseDN, params)

	ctx, cancel := context.WithTimeout(context.Background(), django.ConfigGet(
		django.Global.Settings,
		APPVAR_LDAP_TIMEOUT,
		DEFAULT_LDAP_TIMEOUT,
	))
	defer cancel()

	// 2. Route the request based on the parameters Postfix sent
	if alias := params["othermailbox"]; alias != "" {
		searchAlias(ctx, w, baseDN, alias)
	} else if mail := params["mail"]; strings.HasPrefix(mail, "*@") {
		searchDomain(ctx, w, baseDN, strings.TrimPrefix(mail, "*@"))
	} else if mail != "" {
		searchUser(ctx, w, baseDN, mail)
	} else {
		_log.Warnf("[SEARCH] Unhandled query parameters: %v", params)
	}

	// Always conclude the search operation
	w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
}

func searchAlias(ctx context.Context, w ldapserver.ResponseWriter, baseDN, aliasMail string) {
	aliasRow, err := queries.GetQuerySetWithContext(ctx, &mailmgmt.MailAlias{}).
		Filter("Email__iexact", aliasMail).
		Filter("IsActive", true).
		First()

	if err != nil || aliasRow == nil || aliasRow.Object == nil {
		_log.Warnf("Alias is nil or an error occurred: %v", err)
		w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
		return
	}

	alias := aliasRow.Object
	entry := ldapserver.NewSearchResultEntry("cn=" + alias.Email.Address + "," + baseDN)

	entry.AddAttribute(
		message.AttributeDescription("objectClass"),
		message.AttributeValue("user"),
		message.AttributeValue("alias"),
	)
	entry.AddAttribute(
		message.AttributeDescription("otherMailbox"),
		message.AttributeValue(alias.Email.Address),
	)

	userRows, err := alias.Destination.Objects().WithContext(ctx).All()
	if err != nil || len(userRows) == 0 {
		_log.Warnf("Users are not found for alias or an error occurred: %v", err)
	}

	var mailValues []message.AttributeValue
	for u := range userRows.Objects() {
		mailValues = append(mailValues, message.AttributeValue(u.Email.Address))
	}

	if len(mailValues) > 0 {
		entry.AddAttribute(message.AttributeDescription("mail"), mailValues...)
	}

	w.Write(entry)
}

func searchDomain(ctx context.Context, w ldapserver.ResponseWriter, baseDN, domainName string) {
	domainRow, err := queries.GetQuerySetWithContext(ctx, &mailmgmt.Domain{}).
		Filter("Domain__iexact", domainName).
		First()

	if err != nil || domainRow == nil || domainRow.Object == nil {
		_log.Warnf("Domain is nil or an error occurred: %v", err)
		w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
		return
	}

	domain := domainRow.Object
	entry := ldapserver.NewSearchResultEntry("cn=" + domain.Domain + "," + baseDN)

	entry.AddAttribute(
		message.AttributeDescription("objectClass"),
		message.AttributeValue("top"),
		message.AttributeValue("domainRelatedObject"),
	)

	entry.AddAttribute(
		message.AttributeDescription("cn"),
		message.AttributeValue(domain.Domain),
	)

	entry.AddAttribute(
		message.AttributeDescription("associatedDomain"),
		message.AttributeValue(domain.Domain),
	)

	entry.AddAttribute(
		message.AttributeDescription("mailEnabled"),
		message.AttributeValue("TRUE"),
	)
	w.Write(entry)
}

func searchUser(ctx context.Context, w ldapserver.ResponseWriter, baseDN, email string) {
	userRow, err := queries.GetQuerySet(&auth.User{}).
		WithContext(ctx).
		Select("*", "Profile.*").
		Filter("Email__iexact", email).
		Filter("IsActive", true).
		Get()

	if err != nil || userRow == nil || userRow.Object == nil {
		_log.Warnf("User is nil or an error occurred: %v", err)
		w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
		return
	}

	user := userRow.Object
	entry := ldapserver.NewSearchResultEntry("uid=" + user.Username + "," + baseDN)
	entry.AddAttribute(
		message.AttributeDescription("objectClass"),
		message.AttributeValue("user"),
		message.AttributeValue("person"),
	)
	entry.AddAttribute(
		message.AttributeDescription("uid"),
		message.AttributeValue(user.Username),
	)

	if user.Email != nil {
		entry.AddAttribute(
			message.AttributeDescription("mail"),
			message.AttributeValue(user.Email.Address),
		)
	}

	if user.FirstName != "" || user.LastName != "" {
		entry.AddAttribute(
			message.AttributeDescription("cn"),
			message.AttributeValue(strings.TrimSpace(user.FirstName+" "+user.LastName)),
		)
	} else {
		entry.AddAttribute(
			message.AttributeDescription("cn"),
			message.AttributeValue(user.Username),
		)
	}

	profile, ok := user.FieldDefs().Get("Profile").(*mailmgmt.UserMailProfile)
	if ok && profile != nil && profile.Bytes > 0 {
		entry.AddAttribute(
			message.AttributeDescription("mailQuota"),
			message.AttributeValue(profile.DovecotQuota()),
		)
	}

	w.Write(entry)
}

// flattenFilterAST walks the binary AST once and extracts all queried attributes
// into a simple, easy-to-read map.
// Example: (&(objectClass=user)(mail=nigel@go-dev.nl)) -> map["objectclass":"user", "mail":"nigel@go-dev.nl"]
func flattenFilterAST(f message.Filter, out map[string]string) {
	switch ft := f.(type) {
	case message.FilterEqualityMatch:
		out[strings.ToLower(string(ft.AttributeDesc()))] = string(ft.AssertionValue())
	case message.FilterSubstrings:
		attr := strings.ToLower(string(ft.Type_()))
		for _, sub := range ft.Substrings() {
			if finalStr, ok := sub.(message.SubstringFinal); ok {
				out[attr] = "*" + string(finalStr)
			}
			if anyStr, ok := sub.(message.SubstringAny); ok {
				out[attr] = "*" + string(anyStr)
			}
		}
	case message.FilterAnd:
		for _, subFilter := range ft {
			flattenFilterAST(subFilter, out)
		}
	case message.FilterOr:
		for _, subFilter := range ft {
			flattenFilterAST(subFilter, out)
		}
	}
}
