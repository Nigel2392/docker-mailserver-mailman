package mailmgmt

import (
	"context"
	"errors"
	"fmt"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/mux"
	"github.com/moby/moby/client"
)

const (
	MAILSERVER_CONTAINER_NAME = "mailmgmt.MAILSERVER_CONTAINER_NAME"
)

var CONFIG *MailManagementConfig

type MailManagementConfig struct {
	*apps.AppConfig
	Docker                  *client.Client
	MailServerContainerName string
}

func Setup() *SetupCommand {
	return SetupCtx(context.Background())
}

func SetupCtx(ctx context.Context) *SetupCommand {
	return CONFIG.CommandSetup(ctx)
}

func NewAppConfig() django.AppConfig {
	CONFIG = &MailManagementConfig{
		AppConfig: apps.NewAppConfig("mailmgmt"),
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

		return nil
	}

	CONFIG.Routing = func(m mux.Multiplexer) {
		var group = m.Any("", mux.NewHandler(CONFIG.ViewIndex), "mailmgmt")
		//group.Use(authentication.LoginRequiredMiddleware(func(w http.ResponseWriter, r *http.Request) {
		//	http.Redirect(w, r, django.Reverse("auth:login"), 302)
		//}))

		group.Get("/emails", mux.NewHandler(CONFIG.ViewEmails), "emails")
	}

	return CONFIG
}

func (c *MailManagementConfig) CommandSetup(ctx context.Context) *SetupCommand {
	var setup = &SetupCommand{
		ctx:  ctx,
		c:    c,
		args: make([]string, 0),
	}
	setup.args = append(setup.args, "setup")
	return setup
}
