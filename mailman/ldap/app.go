package ldap

import (
	"errors"
	"fmt"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/goldcrest"
	"github.com/vjeantet/ldapserver"
)

type AppConfig struct {
	*apps.DBRequiredAppConfig
	Server *ldapserver.Server
}

var LDAP *AppConfig

const (
	APPVAR_LDAP_PORT = "ldap.APPVAR_LDAP_PORT"
	APPVAR_LDAP_HOST = "ldap.APPVAR_LDAP_HOST"

	DEFAULT_LDAP_PORT = "3890"
	DEFAULT_LDAP_HOST = "127.0.0.1"
)

func NewAppConfig() django.AppConfig {
	LDAP = &AppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig("ldap"),
	}

	LDAP.Deps = []string{
		"auth",
	}

	LDAP.ModelObjects = []attrs.Definer{
		// &MailAliasUser{},
		&MailAlias{},
		&Domain{},
	}

	LDAP.Server = ldapserver.NewServer()
	LDAP.Server.ReadTimeout = time.Second * 5
	LDAP.Server.WriteTimeout = time.Second * 5

	routes := ldapserver.NewRouteMux()
	routes.Bind(handleBind)
	routes.Search(handleSearch)
	LDAP.Server.Handle(routes)

	goldcrest.Register(django.HOOK_SERVER_STARTUP, 0, django.DjangoHook(func(a *django.Application) error {
		ldapPort := django.ConfigGet(
			django.Global.Settings,
			APPVAR_LDAP_PORT,
			DEFAULT_LDAP_PORT,
		)

		ldapHost := django.ConfigGet(
			django.Global.Settings,
			APPVAR_LDAP_HOST,
			DEFAULT_LDAP_HOST,
		)

		go func() {
			logger.Infof("Serving LDAP on  %s:%s...", ldapHost, ldapPort)

			if err := LDAP.Server.ListenAndServe(fmt.Sprintf("%s:%s", ldapHost, ldapPort)); err != nil {
				err2 := django.Global.Quit()
				if err2 != nil {
					err = errors.Join(err, err2)
				}
				logger.Fatal(1, err)
			}
		}()

		return nil
	}))

	goldcrest.Register(django.HOOK_SERVER_SHUTDOWN, 0, django.DjangoHook(func(a *django.Application) error {
		LDAP.Server.Stop()
		return nil
	}))

	return LDAP
}
