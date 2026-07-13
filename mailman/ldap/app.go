package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
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

	APPVAR_LDAP_TLS_ENABLED              = "ldap.APPVAR_LDAP_TLS_ENABLED" // bool
	APPVAR_LDAP_TLS_CERT_FILE            = "ldap.APPVAR_LDAP_TLS_CERT_FILE"
	APPVAR_LDAP_TLS_KEY_FILE             = "ldap.APPVAR_LDAP_TLS_KEY_FILE"
	APPVAR_LDAP_TLS_INSECURE_SKIP_VERIFY = "ldap.APPVAR_LDAP_TLS_INSECURE_SKIP_VERIFY" // bool
	APPVAR_LDAP_TIMEOUT                  = "ldap.APPVAR_LDAP_TIMEOUT"                  // time.Duration

	DEFAULT_LDAP_PORT    = "3890"
	DEFAULT_LDAP_HOST    = "0.0.0.0"
	DEFAULT_LDAP_TIMEOUT = time.Second * 300

	DEFAULT_LDAP_TLS_ENABLED              = false
	DEFAULT_LDAP_TLS_INSECURE_SKIP_VERIFY = false
	DEFAULT_LDAP_TLS_CERT                 = ""
	DEFAULT_LDAP_TLS_KEY                  = ""
)

func NewAppConfig() django.AppConfig {
	LDAP = &AppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig("ldap"),
	}

	LDAP.Deps = []string{
		"auth",
	}

	LDAP.ModelObjects = []attrs.Definer{}

	LDAP.Server = ldapserver.NewServer()
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
			django.APPVAR_HOST,
			DEFAULT_LDAP_HOST,
		)

		tlsEnabled := django.ConfigGet(
			django.Global.Settings,
			APPVAR_LDAP_TLS_ENABLED,
			DEFAULT_LDAP_TLS_ENABLED,
		)

		certFile := django.ConfigGet(
			django.Global.Settings,
			APPVAR_LDAP_TLS_CERT_FILE,
			DEFAULT_LDAP_TLS_CERT,
		)

		keyFile := django.ConfigGet(
			django.Global.Settings,
			APPVAR_LDAP_TLS_KEY_FILE,
			DEFAULT_LDAP_TLS_KEY,
		)

		timeout := django.ConfigGet(
			django.Global.Settings,
			APPVAR_LDAP_TIMEOUT,
			DEFAULT_LDAP_TIMEOUT,
		)

		LDAP.Server.ReadTimeout = timeout
		LDAP.Server.WriteTimeout = timeout

		var addr = fmt.Sprintf("%s:%s", ldapHost, ldapPort)

		go func() {

			var err error
			if tlsEnabled {
				logger.Infof("Serving LDAPS on %s...", addr)

				cert, certErr := tls.LoadX509KeyPair(certFile, keyFile)
				if certErr != nil {
					logger.Fatal(1, fmt.Errorf("failed to load TLS keys: %w", certErr))
					return
				}

				LDAP.Server.TLSConfig = &tls.Config{
					Certificates: []tls.Certificate{cert},
					InsecureSkipVerify: django.ConfigGet(
						django.Global.Settings,
						APPVAR_LDAP_TLS_INSECURE_SKIP_VERIFY,
						DEFAULT_LDAP_TLS_INSECURE_SKIP_VERIFY,
					),
				}

				listener, listenErr := net.Listen("tcp", addr)
				if listenErr != nil {
					logger.Fatal(1, fmt.Errorf("failed to create TCP listener: %w", listenErr))
					return
				}

				err = LDAP.Server.ServeTLS(listener)
			} else {
				logger.Infof("Serving LDAP on %s...", addr)
				err = LDAP.Server.ListenAndServe(addr)
			}
			if err != nil {
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
