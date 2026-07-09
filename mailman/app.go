package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/components"
	cmpts "github.com/Nigel2392/go-django/src/contrib/admin/components"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/staticfiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/mux"
	"github.com/a-h/templ"
)

func init() {
	if len(modes) <= 1 {
		panic("there must be more than one mode, dummy!")
	}

	modes[len(modes)-1].Next = &modes[0]

	//len -2 so modes[i]+1 is safe.
	for i := 0; i < len(modes)-1; i++ {
		modes[i].Next = &modes[i+1]
	}

	for _, color := range modes {
		_modes[color.Name] = color

		assert.True(
			color.Next != nil,
			"logic fail, start debugging: %+v", color,
		)
	}
}

func NewAppConfig() django.AppConfig {
	var app = apps.NewAppConfig("main")
	var tplFS, staticFS = initAppFS()

	app.Routing = func(m mux.Multiplexer) {
		m.Use(ModeMiddleware)
	}

	tpl.RequestFuncs(func(r *http.Request) template.FuncMap {
		return template.FuncMap{
			"mode": func() *boundColors {
				return modeFromContext(r)
			},
			"modes": func() []string {
				var l = make([]string, 0, len(modes))
				for _, mode := range modes {
					l = append(l, mode.Name)
				}
				return l
			},
		}
	})

	app.TemplateConfig = &tpl.Config{
		AppName: "main",
		FS:      tplFS,
		Bases: []string{
			"main/base/skeleton.tmpl",
			"main/base/left.tmpl",
			"main/base/right.tmpl",
			"main/base/navbar.tmpl",
			"main/base/messages.tmpl",

			"mailmgmt/base/delete_form.tmpl",
			"mailmgmt/partials/list_form.tmpl",
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

	tpl.Add(tpl.Config{
		AppName: "auth",
		FS:      tplFS,
		Bases: []string{
			"auth/base.tmpl",
			"main/base/messages.tmpl",
		},
		Matches: filesystem.MatchOr(
			filesystem.MatchAnd(
				filesystem.MatchPrefix("auth/"),
				filesystem.MatchExt(".tmpl"),
			),
		),
	})

	components.Register("header", func(level int, headingText, subText string, extra ...any) templ.Component {
		var comps []templ.Component
		for _, c := range extra {
			switch v := c.(type) {
			case templ.Component:
				comps = append(comps, v)
			case []templ.Component:
				if comps == nil {
					comps = v
				} else {
					comps = append(comps, v...)
				}
			default:
				panic(fmt.Sprintf(
					"unexpected component type: %T", c,
				))
			}
		}
		return cmpts.Header(level, headingText, subText, comps...)
	})

	app.Init = func(settings django.Settings) error {

		// Set up the static files for this app
		// They are stored in the "assets/static" directory
		staticfiles.AddFS(staticFS, nil)

		return nil
	}

	return app
}

type boundColors struct {
	Colors
	r *http.Request
}

func (bc boundColors) Label() string {
	return bc.Colors.Label(bc.r.Context())
}

func (bc boundColors) Next() string {
	return bc.Colors.Next.Label(bc.r.Context())
}

type Colors struct {
	Name          string
	Label         func(context.Context) string
	Next          *Colors
	Primary       string
	PrimaryAccent string
	Secondary     string
	StandOut      string
	BorderStyle   string // dotted, solid, dashed
	Header        string
	HeaderText    string
	LogoURL       string
}

var _modes = make(map[string]Colors)
var modes = []Colors{
	{
		Name:          __light_mode,
		Label:         trans.S("Light"),
		BorderStyle:   "solid",
		Primary:       "27, 27, 27",
		PrimaryAccent: "20, 163, 156",
		StandOut:      "20, 163, 156",
		Secondary:     "255, 255, 255",
		Header:        "20, 163, 156",
		HeaderText:    "255, 255, 255",
		LogoURL:       "/static/mailman.png",
	},
	{
		Name:          "dark",
		Label:         trans.S("Dark"),
		BorderStyle:   "dotted",
		Primary:       "255, 255, 255",
		PrimaryAccent: "124, 124, 124",
		StandOut:      "20, 163, 156",
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
		return &boundColors{_modes[__light_mode], r}
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
			var mode, ok = _modes[modeCookie.Value]
			if !ok {
				goto lightMode
			}

			r = modeToContext(r, mode)

			next.ServeHTTP(w, r)
			return
		}

	lightMode:
		r = modeToContext(r, _modes[__light_mode])

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
