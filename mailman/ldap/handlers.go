package ldap

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	queries "github.com/Nigel2392/go-django/queries/src"
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
	//		log.Printf("[BIND] Rate limit exceeded/blocked for IP: %s", m.Client.Addr().String())
	//		res := ldapserver.NewBindResponse(ldapserver.LDAPResultBusy)
	//		res.SetDiagnosticMessage("Too many attempts. Try again later.")
	//		w.Write(res)
	//		return
	//	}

	r := m.GetBindRequest()
	bindDN := string(r.Name())
	password := string(r.AuthenticationSimple())

	log.Printf("[BIND] Attempt: DN=%s", bindDN)

	if len(password) > 64 || len(password) < 4 || strings.TrimSpace(password) == "" {
		log.Printf("[BIND] Rejected, DN=%s, PasswordLen=%d", bindDN, len(password))
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultUnwillingToPerform)
		res.SetDiagnosticMessage("Authentication not aligned with standards.")
		w.Write(res)
		return
	}

	parsedDN, err := ldap.ParseDN(bindDN)
	if err != nil || len(parsedDN.RDNs) == 0 {
		log.Printf("[BIND] Malformed DN rejected: %v", err)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		log.Printf("[BIND] Unsupported bind attribute: %s", attrType)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	userRow, err := qs.Get()
	if err != nil {
		log.Printf("[BIND] Failed to retrieve %s=%s (%v)", attrType, attrValue, err)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	u := userRow.Object
	if !u.IsActive {
		log.Printf("[BIND] Access denied: User %s is inactive", u.Username)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	if err := u.Password.Check(password); err != nil {
		log.Printf("[BIND] Invalid password for user: %s", u.Username)
		res := ldapserver.NewBindResponse(ldapserver.LDAPResultInvalidCredentials)
		w.Write(res)
		return
	}

	// Session established securely!
	// _ = RateLimit.Reset(ctx, m)
	m.Client.SetData(u.IsAdministrator)

	log.Printf("[BIND] SUCCESS for user: %s", u.Username)
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
		log.Println("[SEARCH] Rejected unauthorized search attempt")
		w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultInsufficientAccessRights))
		return
	}

	r := m.GetSearchRequest()
	baseDN := string(r.BaseObject())

	// 1. Flatten the AST into a map
	params := make(map[string]string)
	flattenFilterAST(r.Filter(), params)

	log.Printf("[SEARCH] Base: %s | Params: %v", baseDN, params)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Route the request based on the parameters Postfix sent
	if alias := params["othermailbox"]; alias != "" {
		searchAlias(ctx, w, baseDN, alias)
	} else if mail := params["mail"]; strings.HasPrefix(mail, "*@") {
		searchDomain(ctx, w, baseDN, strings.TrimPrefix(mail, "*@"))
	} else if mail != "" {
		searchUser(ctx, w, baseDN, mail)
	} else {
		log.Printf("[SEARCH] Unhandled query parameters: %v", params)
	}

	// Always conclude the search operation
	w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
}

func searchAlias(ctx context.Context, w ldapserver.ResponseWriter, baseDN, aliasMail string) {
	aliasRow, err := queries.GetQuerySetWithContext(ctx, &mailmgmt.MailAlias{}).
		Filter("Source__iexact", aliasMail).
		Filter("IsActive", true).
		First()

	if err != nil || aliasRow == nil || aliasRow.Object == nil {
		return
	}

	alias := aliasRow.Object
	entry := ldapserver.NewSearchResultEntry("cn=" + alias.Source.Address + "," + baseDN)

	entry.AddAttribute(
		message.AttributeDescription("objectClass"),
		message.AttributeValue("user"),
		message.AttributeValue("alias"),
	)
	entry.AddAttribute(
		message.AttributeDescription("otherMailbox"),
		message.AttributeValue(alias.Source.Address),
	)

	userRows, _ := alias.Destination.Objects().WithContext(ctx).All()
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
		return
	}

	domain := domainRow.Object
	entry := ldapserver.NewSearchResultEntry("dc=" + domain.Domain + "," + baseDN)
	entry.AddAttribute(
		message.AttributeDescription("objectClass"),
		message.AttributeValue("domain"),
	)
	entry.AddAttribute(
		message.AttributeDescription("dc"),
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
		Filter("Profile.Deleted", false).
		Get()

	if err != nil || userRow == nil {
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

// walkASTForAttribute recursively evaluates the binary AST constructed
// by vjeantet/goldap. It completely bypasses string serialization issues.
func walkASTForAttribute(f message.Filter, targetAttr string) string {
	switch ft := f.(type) {
	case message.FilterEqualityMatch:
		// Base case: (mail=nigel@example.com)
		if strings.EqualFold(string(ft.AttributeDesc()), targetAttr) {
			return string(ft.AssertionValue())
		}
	case message.FilterAnd:
		// Logical AND: (&(objectClass=user)(mail=nigel@...))
		for _, subFilter := range ft {
			if val := walkASTForAttribute(subFilter, targetAttr); val != "" {
				return val
			}
		}
	case message.FilterOr:
		// Logical OR: (|(mail=nigel)(uid=nigel))
		for _, subFilter := range ft {
			if val := walkASTForAttribute(subFilter, targetAttr); val != "" {
				return val
			}
		}
	case message.FilterSubstrings:
		// Substrings: (mail=*@example.com)
		if strings.EqualFold(string(ft.Type_()), targetAttr) {
			for _, sub := range ft.Substrings() {
				if finalStr, ok := sub.(message.SubstringFinal); ok {
					return "*@" + string(finalStr)
				}
				if anyStr, ok := sub.(message.SubstringAny); ok {
					return "*@" + string(anyStr)
				}
			}
		}
	}
	return ""
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
				out[attr] = "*@" + string(finalStr)
			}
			if anyStr, ok := sub.(message.SubstringAny); ok {
				out[attr] = "*@" + string(anyStr)
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
