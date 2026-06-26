package sieve

import (
	"errors"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

const MAILMAN_SIEVE_TEMPLATE = "sieve.MAILMAN_SIEVE_TEMPLATE"

type SieveAppConfig struct {
	*apps.DBRequiredAppConfig
	_enabled bool
}

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

	_app.Init = func(settings django.Settings, db drivers.Database) error {

		return nil
	}

	return _app
}
