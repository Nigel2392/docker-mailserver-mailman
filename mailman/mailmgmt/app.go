package mailmgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Nigel2392/cache"
	"github.com/Nigel2392/docker-mailserver-mailman/mailman/docker"
	mailmgmt_cache "github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt/cache"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const (
	_MAILSERVER_CONTAINER_INSPECT_CACHE_KEY = "mailmgmt.MAILSERVER_INSPECT_RESULT"
	MAILSERVER_CONTAINER_NAME               = "mailmgmt.MAILSERVER_CONTAINER_NAME"
	MAILSERVER_CACHING_ENABLED              = "mailmgmt.MAILSERVER_CACHING_ENABLED"
	EMAIL_REGEX                             = `([a-zA-Z0-9_.+,"-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)`
)

var CONFIG *MailManagementConfig

type MailManagementConfig struct {
	*apps.AppConfig
	//pool                    *shell.ExecPool
}

var _, _ = queries.SignalPostModelCreate.Listen(func(s signals.Signal[queries.SignalSave], ss queries.SignalSave) (err error) {
	switch i := ss.Instance.(type) {
	case *auth.User:
		_, _, err = queries.
			GetQuerySetWithContext(ss.Context, &UserMailProfile{}).
			Filter("User", i).
			GetOrCreate(&UserMailProfile{
				User: i,
			})
	}
	return err
})

func NewAppConfig() django.AppConfig {

	CONFIG = &MailManagementConfig{
		AppConfig: apps.NewAppConfig("mailmgmt"),
	}

	CONFIG.ModelObjects = []attrs.Definer{
		&MailAlias{},
		&MailAliasUser{},
		&UserMailProfile{},
		&Domain{},
	}

	CONFIG.Init = func(settings django.Settings) (err error) {

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
		goldcrest.Register(django.HOOK_SERVER_ERROR, 0, django.ServerErrorHook(func(w http.ResponseWriter, r *http.Request, app *django.Application, err except.ServerError) {

		}))
		return nil
	}

	CONFIG.Routing = func(m mux.Multiplexer) {
		var group = m.Any("", mux.NewHandler(CONFIG.ViewIndex), "mailmgmt")
		group.Use(authentication.LoginRequiredMiddleware(func(w http.ResponseWriter, r *http.Request) {
			autherrors.Fail(http.StatusUnauthorized, "you need to be logged in.")
		}))
		var htmx = group.Get("/htmx", nil, "htmx")
		//group.Use(authentication.LoginRequiredMiddleware(func(w http.ResponseWriter, r *http.Request) {
		//	http.Redirect(w, r, django.Reverse("auth:login"), 302)
		//}))

		emails := group.Get("/emails", views.Serve(ViewEmails), "emails")
		emails.Get("/delete", views.Serve(ViewDeleteEmail), "delete")
		emails.Post("/delete", views.Serve(ViewDeleteEmail))

		htmxEmails := htmx.Get("/emails", nil, "emails")
		htmxEmails.Get("/add", views.Serve(ViewAddEmailHtmx), "add")
		htmxEmails.Post("/add", views.Serve(ViewAddEmailHtmx))
		htmxEmails.Get("/update", views.Serve(ViewUpdateEmailPasswordHtmx), "update")
		htmxEmails.Post("/update", views.Serve(ViewUpdateEmailPasswordHtmx))

		aliasses := group.Get("/aliasses", views.Serve(ViewAliasses), "aliasses")
		_ = aliasses

		htmxAliases := htmx.Get("/aliasses", nil, "aliasses")
		htmxAliases.Get("/add", views.Serve(ViewAddAliasHtmx), "add")
		htmxAliases.Post("/add", views.Serve(ViewAddAliasHtmx))

		domains := group.Get("/domains", views.Serve(ViewDomains), "domains")
		domains.Get("/add", views.Serve(ViewAddDomain), "add")
		domains.Post("/add", views.Serve(ViewAddDomain), "add")
		domains.Get("/delete", views.Serve(ViewDeleteDomain), "delete")
		domains.Post("/delete", views.Serve(ViewDeleteDomain), "delete")
	}

	return CONFIG
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

func MailServer(ctx context.Context, refresh bool) (*container.InspectResponse, error) {
	cli, err := docker.DockerErr()
	if err != nil {
		return nil, err
	}

	var (
		result  container.InspectResponse
		rawData []byte
		cacheV  any
		ok      bool
	)

	if refresh {
		goto notCached
	}

	cacheV, err = cache.Get(ctx, _MAILSERVER_CONTAINER_INSPECT_CACHE_KEY)
	if err != nil && !errors.Is(err, cache.ErrItemNotFound) {
		return nil, err
	}

	rawData, ok = cacheV.([]byte)
	if !ok || len(rawData) == 0 {
		goto notCached
	}

	err = json.Unmarshal(rawData, &result)
	if err != nil {
		return nil, err
	}

notCached:
	containerName := django.ConfigGet(
		django.Global.Settings,
		MAILSERVER_CONTAINER_NAME,
		"mailserver",
	)

	res, err := cli.ContainerInspect(
		ctx, containerName,
		client.ContainerInspectOptions{},
	)
	if err != nil {
		return nil, ErrDocker.WithCause(fmt.Errorf(
			"could not retrieve container %q, are you sure it is running? %w",
			containerName, err,
		))
	}

	cache.Set(
		ctx, _MAILSERVER_CONTAINER_INSPECT_CACHE_KEY, []byte(res.Raw), time.Minute*5,
	)

	return &res.Container, nil
}

var _matchEmail = regexp.MustCompile(EMAIL_REGEX)

func IsValidEmail(email string) bool {
	return _matchEmail.MatchString(email)
}
