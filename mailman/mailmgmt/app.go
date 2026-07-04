package mailmgmt

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Nigel2392/cache"
	mailmgmt_cache "github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt/cache"
	merrs "github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/goldcrest"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/moby/moby/client"
)

const (
	MAILSERVER_CONTAINER_NAME  = "mailmgmt.MAILSERVER_CONTAINER_NAME"
	MAILSERVER_CACHING_ENABLED = "mailmgmt.MAILSERVER_CACHING_ENABLED"
	EMAIL_REGEX                = `([a-zA-Z0-9_.+,"-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)`
)

var CONFIG *MailManagementConfig

type MailManagementConfig struct {
	*apps.AppConfig
	Docker                  *client.Client
	MailServerContainerName string
	res                     *client.ContainerInspectResult
	//pool                    *shell.ExecPool
}

func NewAppConfig() django.AppConfig {

	CONFIG = &MailManagementConfig{
		AppConfig: apps.NewAppConfig("mailmgmt"),
	}

	CONFIG.ModelObjects = []attrs.Definer{
		&MailAlias{},
		&UserMailQuota{},
	}

	CONFIG.Init = func(settings django.Settings) (err error) {
		var ok bool
		CONFIG.MailServerContainerName, ok = django.ConfigGetOK[string](
			settings, MAILSERVER_CONTAINER_NAME, "mailserver",
		)
		if !ok || CONFIG.MailServerContainerName == "" {
			return errors.New("no mailserver container name configured")
		}

		// Set up docker client
		ctx := context.Background()
		CONFIG.Docker, err = client.New(
			client.FromEnv,
			client.WithTimeout(time.Second*10),
		)
		if err != nil {
			return err
		}

		// Check mailserver exists
		_, err = CONFIG.Docker.ContainerInspect(
			ctx, CONFIG.MailServerContainerName,
			client.ContainerInspectOptions{},
		)
		if err != nil {
			return fmt.Errorf(
				"could not retrieve container %q, are you sure it is running? %w",
				CONFIG.MailServerContainerName, err,
			)
		}

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
			if !merrs.IsMailserverError(err) {
				return
			}

			var mErr = new(merrs.MailserverError)
			if !errors.As(err, mErr) {
				panic(err)
			}

			switch mErr.Code {
			case merrs.CodeUnknown:
			case merrs.CodeNotRunning:
			}

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

		aliases := group.Get("/aliases", views.Serve(ViewEmails), "aliases")
		_ = aliases

		htmxAliases := htmx.Get("/alias", nil, "alias")
		htmxAliases.Get("/add", views.Serve(ViewAddAliasHtmx), "add")
		htmxAliases.Post("/add", views.Serve(ViewAddAliasHtmx))
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

func (c *MailManagementConfig) InspectDockerMailServer(ctx context.Context, size bool) (client.ContainerInspectResult, error) {
	if c.res != nil {
		return *c.res, nil
	}
	var res, err = c.Docker.ContainerInspect(
		ctx, c.MailServerContainerName, client.ContainerInspectOptions{Size: size},
	)
	c.res = &res
	return res, err
}

var _matchEmail = regexp.MustCompile(EMAIL_REGEX)

func IsValidEmail(email string) bool {
	return _matchEmail.MatchString(email)
}
