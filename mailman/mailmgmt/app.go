package mailmgmt

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/Nigel2392/cache"
	"github.com/Nigel2392/docker-mailserver-mailman/mailman/chooser"
	mailmgmt_cache "github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt/cache"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/admin"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	bytesize "github.com/inhies/go-bytesize"
)

const (
	_MAILSERVER_CONTAINER_INSPECT_CACHE_KEY = "MAILSERVER_INSPECT_RESULT"
	MAILSERVER_CONTAINER_NAME               = "MAILSERVER_CONTAINER_NAME"
	MAILSERVER_DEFAULT_USER_QUOTA           = "MAILSERVER_DEFAULT_USER_QUOTA"
	MAILSERVER_CACHING_ENABLED              = "MAILSERVER_CACHING_ENABLED"

	MAILMAN_DOVEADM_API_ADDR = "MAILMAN_DOVEADM_API_ADDR"
	MAILMAN_DOVEADM_API_KEY  = "MAILMAN_DOVEADM_API_KEY"

	EMAIL_REGEX = `([a-zA-Z0-9_.+,"-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)`
)

var CONFIG *MailManagementConfig

type MailManagementConfig struct {
	*apps.AppConfig
	defaultInboxSize uint
	//pool                    *shell.ExecPool
}

type DetailObject[OBJECT any, FORM forms.Form] struct {
	Object OBJECT
	Form   FORM
}

func (d *DetailObject[OBJECT, FORM]) String() string {
	if s, ok := any(d.Object).(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%v", d.Object)
}

var _, _ = queries.SignalPostModelCreate.Listen(func(s signals.Signal[queries.SignalSave], ss queries.SignalSave) (err error) {
	switch i := ss.Instance.(type) {
	case *auth.User:

		_, _, err = queries.
			GetQuerySetWithContext(ss.Context, &UserMailProfile{}).
			Filter("User", i).
			GetOrCreate(&UserMailProfile{
				User:  i,
				Bytes: CONFIG.defaultInboxSize,
			})
	}
	return err
})

func NewAppConfig() django.AppConfig {

	attrs.RegisterConfigChange(&auth.User{}, "Username", func(fc *attrs.FieldConfig) {
		fc.MaxLength = 256
	})

	CONFIG = &MailManagementConfig{
		AppConfig: apps.NewAppConfig("mailmgmt"),
	}

	CONFIG.ModelObjects = []attrs.Definer{
		&MailAlias{},
		&MailAliasUser{},
		&UserMailProfile{},
		&Domain{},
		&ServiceAccount{},
	}

	CONFIG.Init = func(settings django.Settings) (err error) {

		var bytesStr = django.ConfigGet(
			django.Global.Settings,
			MAILSERVER_DEFAULT_USER_QUOTA,
			"5GB",
		)

		bytes, err := bytesize.Parse(bytesStr)
		if err != nil {
			return err
		}

		CONFIG.defaultInboxSize = uint(bytes)

		//	if !inspectResult.Container.State.Running {
		//
		//	}
		//
		//	_, pool, err := shell.StartPool(context.Background(), CONFIG.Docker, CONFIG.MailServerContainerName)
		//	if err != nil {
		//		return fmt.Errorf("could not start exec pool: %w", err)
		//	}
		//	CONFIG.pool = pool
		return nil
	}

	CONFIG.Ready = func() error {
		chooser.Register(&chooser.ChooserDefinition[*auth.User]{
			ChooserKey: "mailman_user",
			Title:      trans.S("User Chooser"),
			Model:      &auth.User{},
			PreviewString: func(ctx context.Context, instance *auth.User) string {
				return instance.Email.Address
			},
			ListPage: &chooser.ChooserListPage[*auth.User]{
				Fields: []string{
					"Email",
					"FirstName",
					"LastName",
					"IsActive",
				},
				SearchFields: []admin.SearchField{
					{
						Name:   "Username",
						Lookup: expr.LOOKUP_ICONTANS,
					},
					{
						Name:   "Email",
						Lookup: expr.LOOKUP_ICONTANS,
					},
					{
						Name:   "FirstName",
						Lookup: expr.LOOKUP_ICONTANS,
					},
					{
						Name:   "LastName",
						Lookup: expr.LOOKUP_ICONTANS,
					},
				},
				QuerySet: func(r *http.Request, model *auth.User) (*queries.QuerySet[*auth.User], error) {
					var aliasId = r.URL.Query().Get("alias_id")
					var excl = r.URL.Query().Get("exclude") != ""
					var qs = queries.GetQuerySet(&auth.User{})
					if aliasId != "" {
						var throughModelQs = queries.GetQuerySet(&MailAliasUser{}).Filter("AliasID", aliasId)
						qs = qs.Filter(expr.Q("ID__in", queries.Subquery(throughModelQs.Select("UserID"))).Not(excl)).Distinct()
					}
					return qs.OrderBy("Email"), nil
				},
			},
		}, "mailman_user")

		chooser.Register(&chooser.ChooserDefinition[*MailAlias]{
			ChooserKey: "mailman_alias",
			Title:      trans.S("Alias Chooser"),
			Model:      &MailAlias{},
			PreviewString: func(ctx context.Context, instance *MailAlias) string {
				return instance.Email.Address
			},
			ExtraData: func(ctx context.Context, instance *MailAlias) map[string]any {
				return map[string]any{"TEST": 1}
			},
			ListPage: &chooser.ChooserListPage[*MailAlias]{
				Fields: []string{
					"Email",
					"UserCount",
					"IsActive",
				},
				SearchFields: []admin.SearchField{
					{
						Name:   "Email",
						Lookup: expr.LOOKUP_ICONTANS,
					},
				},
				QuerySet: func(r *http.Request, model *MailAlias) (*queries.QuerySet[*MailAlias], error) {
					var aliasId = r.URL.Query().Get("user_id")
					var excl = r.URL.Query().Get("exclude") != ""
					var qs = queries.
						GetQuerySetWithContext(r.Context(), &MailAlias{}).
						Select("ID", "Email", "IsActive").
						GroupBy("ID").
						Annotate("UserCount", expr.COUNT("Destination.ID"))

					if aliasId != "" {
						var throughModelQs = queries.GetQuerySet(&MailAliasUser{}).Filter("UserID", aliasId)
						qs = qs.Filter(expr.Q("ID__in", queries.Subquery(throughModelQs.Select("AliasID"))).Not(excl)).Distinct()
					}

					return qs.OrderBy("-IsActive", "-UserCount", "Email"), nil
				},
			},
		}, "mailman_alias")

		goldcrest.Register(django.HOOK_SERVER_ERROR, 0, django.ServerErrorHook(func(w http.ResponseWriter, r *http.Request, app *django.Application, err except.ServerError) {

		}))
		return nil
	}

	CONFIG.Routing = func(m mux.Multiplexer) {
		var group = m.Any("", mux.NewHandler(CONFIG.ViewIndex), "mailmgmt")
		group.Use(
			authentication.LoginRequiredMiddleware(func(w http.ResponseWriter, r *http.Request) {
				autherrors.Fail(http.StatusUnauthorized, "you need to be logged in.")
			}),
			CONFIG.Middleware,
		)

		var htmx = group.Get("/htmx", nil, "htmx")
		//group.Use(authentication.LoginRequiredMiddleware(func(w http.ResponseWriter, r *http.Request) {
		//	http.Redirect(w, r, django.Reverse("auth:login"), 302)
		//}))

		emails := group.Get("/emails", views.Serve(ViewEmails), "emails")
		emails.Get("/delete/<<email_id>>", views.Serve(ViewDeleteEmail), "delete")
		emails.Post("/delete/<<email_id>>", views.Serve(ViewDeleteEmail))
		emails.Get("/detail/<<email_id>>", views.Serve(ViewEmailDetail), "detail")
		emails.Post("/detail/<<email_id>>", views.Serve(ViewEmailDetail))

		htmxEmails := htmx.Any("/emails", nil, "emails")
		htmxEmails.Get("/add", views.Serve(ViewAddEmailHtmx), "add")
		htmxEmails.Post("/add", views.Serve(ViewAddEmailHtmx))
		htmxEmails.Post("/alias/remove/<<email_id>>", views.Serve(ViewEmailAliasRemove), "remove_alias")

		htmxEmails.Get("/update/<<email_id>>", views.Serve(ViewUpdateEmailPasswordHtmx), "update")
		htmxEmails.Post("/update/<<email_id>>", views.Serve(ViewUpdateEmailPasswordHtmx))

		aliasses := group.Get("/aliasses", views.Serve(ViewAliasses), "aliasses")
		aliasses.Get("/detail/<<alias_id>>", views.Serve(ViewAliasDetail), "detail")
		aliasses.Post("/detail/<<alias_id>>", views.Serve(ViewAliasDetail), "detail")
		aliasses.Get("/delete/<<alias_id>>", views.Serve(ViewDeleteAlias), "delete")
		aliasses.Post("/delete/<<alias_id>>", views.Serve(ViewDeleteAlias), "delete")

		htmxAliases := htmx.Get("/aliasses", nil, "aliasses")
		htmxAliases.Get("/add", views.Serve(ViewAddAliasHtmx), "add")
		htmxAliases.Post("/add", views.Serve(ViewAddAliasHtmx))
		htmxAliases.Get("/add/<<email_id>>", views.Serve(ViewAddAliasToUserHtmx), "add_user")
		htmxAliases.Post("/add/<<email_id>>", views.Serve(ViewAddAliasToUserHtmx))
		htmxAliases.Post("/user/remove/<<alias_id>>", views.Serve(ViewAliasRemoveUser), "remove_user")

		domains := group.Get("/domains", views.Serve(ViewDomains), "domains")
		domains.Get("/add", views.Serve(ViewAddDomain), "add")
		domains.Post("/add", views.Serve(ViewAddDomain), "add")
		domains.Get("/delete/<<domain_id>>", views.Serve(ViewDeleteDomain), "delete")
		domains.Post("/delete/<<domain_id>>", views.Serve(ViewDeleteDomain), "delete")
		domains.Get("/disable/<<domain_id>>", views.Serve(ViewDeactivateDomain), "disable")
		domains.Post("/disable/<<domain_id>>", views.Serve(ViewDeactivateDomain), "disable")

		htmxDomains := htmx.Get("/domains", nil, "domains")
		htmxDomains.Post("/activate/<<domain_id>>", views.Serve(ViewActivateDomain), "activate")

		services := group.Get("/services", views.Serve(ViewServiceAccounts), "services")
		services.Get("/add", views.Serve(ViewAddServiceAccount), "add")
		services.Post("/add", views.Serve(ViewAddServiceAccount))
		services.Get("/detail/<<account_id>>", views.Serve(ViewServiceAccountDetail), "detail")
		services.Post("/detail/<<account_id>>", views.Serve(ViewServiceAccountDetail))
		services.Get("/delete/<<account_id>>", views.Serve(ViewDeleteServiceAccount), "delete")
		services.Post("/delete/<<account_id>>", views.Serve(ViewDeleteServiceAccount))

	}

	return CONFIG
}

func (c *MailManagementConfig) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var u = authentication.Retrieve(r)
		if !u.IsAdmin() {
			autherrors.Fail(
				http.StatusForbidden,
				"You are not allowed",
			)
		}

		next.ServeHTTP(w, r)
	})
}

func Cache() *mailmgmt_cache.MailMgmtCache {
	var enabled = django.ConfigGet(
		django.Global.Settings,
		MAILSERVER_CACHING_ENABLED,
		true,
	)

	return mailmgmt_cache.NewMailMgmtCache(
		enabled, cache.Default(),
	)
}

var _matchEmail = regexp.MustCompile(EMAIL_REGEX)

func IsValidEmail(email string) bool {
	return _matchEmail.MatchString(email)
}
