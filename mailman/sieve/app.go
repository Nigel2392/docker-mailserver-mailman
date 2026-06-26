package sieve

import (
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func NewAppConfig() django.AppConfig {
	var app = apps.NewDBAppConfig("sieve")

	app.Deps = []string{
		"mailmgmt",
	}

	app.ModelObjects = []attrs.Definer{
		&BannedEmail{},
		&ForwardedEmail{},
	}

	return app
}
