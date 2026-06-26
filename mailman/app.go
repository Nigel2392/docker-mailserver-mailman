package main

import (
	"context"
	"embed"
	"html/template"
	"net/http"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/mux"
)

//go:embed assets/**
var assetsFS embed.FS

func NewAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("main")

	var (
		tplFS    = filesystem.Sub(assetsFS, "assets/templates")
		staticFS = filesystem.Sub(assetsFS, "assets/static")
	)

	app.Routing = func(m mux.Multiplexer) {
		m.Use(ModeMiddleware)
	}

	tpl.RequestFuncs(func(r *http.Request) template.FuncMap {
		return template.FuncMap{
			"mode": func() *boundColors {
				return modeFromContext(r)
			},
		}
	})

	app.TemplateConfig = &tpl.Config{
		AppName: "main",
		FS:      tplFS,
		Bases: []string{
			"main/base/base.tmpl",
			"main/base/left.tmpl",
			"main/base/right.tmpl",
			"main/base/navbar.tmpl",
			"main/base/messages.tmpl",
		},
		Matches: filesystem.MatchOr(
			filesystem.MatchAnd(
				filesystem.MatchPrefix("main/"),
				filesystem.MatchExt(".tmpl"),
			),
			filesystem.MatchAnd(
				filesystem.MatchPrefix("mailmgmt/"),
				filesystem.MatchExt(".tmpl"),
			),
		),
	}

	app.Init = func(settings django.Settings) error {

		// Set up the static files for this app
		// They are stored in the "assets/static" directory
		staticfiles.AddFS(staticFS, filesystem.MatchAnd(
			filesystem.MatchOr(
				filesystem.MatchExt(".css"),
				filesystem.MatchExt(".js"),
				filesystem.MatchExt(".png"),
				filesystem.MatchExt(".jpg"),
				filesystem.MatchExt(".jpeg"),
				filesystem.MatchExt(".svg"),
				filesystem.MatchExt(".gif"),
				filesystem.MatchExt(".ico"),
			),
		))

		return nil
	}

	return app
}

type boundColors struct {
	Colors
	r *http.Request
}

func (bc boundColors) Name() string {
	return bc.Colors.Name(bc.r.Context())
}

func (bc boundColors) Opposite() string {
	return bc.Colors.Opposite(bc.r.Context())
}

type Colors struct {
	Name          func(context.Context) string
	Opposite      func(context.Context) string
	Primary       string
	PrimaryAccent string
	Secondary     string
	Header        string
	HeaderText    string
	LogoURL       string
}

var modes = map[string]Colors{
	"light": {
		Name:          trans.S("Light"),
		Opposite:      trans.S("Dark"),
		Primary:       "27, 27, 27",
		PrimaryAccent: "20, 163, 156",
		Secondary:     "255, 255, 255",
		Header:        "20, 163, 156",
		HeaderText:    "255, 255, 255",
		LogoURL:       "/static/mailman.png",
	},
	"dark": {
		Name:          trans.S("Dark"),
		Opposite:      trans.S("Light"),
		Primary:       "255, 255, 255",
		PrimaryAccent: "165, 165, 165",
		Secondary:     "27, 27, 27",
		Header:        "27, 27, 27",
		HeaderText:    "255, 255, 255",
		LogoURL:       "/static/mailman.png",
	},
}

type modeContextKey struct{}

const __light_mode = "light"

func modeFromContext(r *http.Request) *boundColors {
	var mode, ok = r.Context().Value(modeContextKey{}).(Colors)
	if !ok {
		return &boundColors{modes[__light_mode], r}
	}
	return &boundColors{mode, r}
}

func modeToContext(r *http.Request, mode Colors) *http.Request {
	return r.WithContext(context.WithValue(
		r.Context(), modeContextKey{}, mode,
	))
}

func ModeMiddleware(next mux.Handler) mux.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var modeCookie, err = r.Cookie("visual-mode")
		if err != nil {
			goto lightMode
		}

		if modeCookie != nil {
			var mode, ok = modes[modeCookie.Value]
			if !ok {
				goto lightMode
			}

			r = modeToContext(r, mode)

			next.ServeHTTP(w, r)
			return
		}

	lightMode:
		r = modeToContext(r, modes[__light_mode])

		// Set the visual mode cookie
		http.SetCookie(w, &http.Cookie{
			Name:    "visual-mode",
			Value:   __light_mode,
			Expires: time.Now().Add(365 * 24 * time.Hour),
			Path:    "/",
		})

		next.ServeHTTP(w, r)
	})
}
