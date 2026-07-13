package sieve

import (
	"embed"
	"errors"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/mux"
)

const MAILMAN_SIEVE_TEMPLATE = "sieve.MAILMAN_SIEVE_TEMPLATE"

type SieveAppConfig struct {
	*apps.DBRequiredAppConfig
	_enabled bool
}

//go:embed assets/**
var assetsFS embed.FS

var (
	_app             *SieveAppConfig
	ErrAppNotEnabled = errors.New("could not initialize app sieve: docker mailserver sieve not enabled")
)

func NewAppConfig() django.AppConfig {
	_app = &SieveAppConfig{
		DBRequiredAppConfig: apps.NewDBAppConfig("sieve"),
	}

	_app.Deps = []string{
		"mailmgmt",
	}
	// CONFIG.Docker.CopyToContainer()

	_app.ModelObjects = []attrs.Definer{
		&BannedEmail{},
		&ForwardedEmail{},
		&VacationRule{},
	}

	_app.TemplateConfig = &tpl.Config{
		FS: filesystem.Sub(assetsFS, "assets/templates"),
		Matches: filesystem.MatchOr(
			filesystem.MatchAnd(
				filesystem.MatchPrefix("sieve/"),
				filesystem.MatchExt(".tmpl"),
			),
		),
	}

	_app.Init = func(settings django.Settings, db drivers.Database) error {

		return nil
	}

	_app.Routing = func(m mux.Multiplexer) {
		sieve := m.Get("/sieve", nil, "sieve")

		forwards := sieve.Get("/forwards", mux.NewHandler(ViewForwardedEmails), "forwards")
		_ = forwards
	}

	return _app
}
