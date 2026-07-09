package mailmgmt

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/views"
)

var (
	_ views.View           = (*DeleteView[any])(nil)
	_ views.MethodsView    = (*DeleteView[any])(nil)
	_ views.BindableView   = (*DeleteView[any])(nil)
	_ views.TemplateKeyer  = (*DeleteView[any])(nil)
	_ views.TemplateGetter = (*DeleteView[any])(nil)
	_ views.View           = (*BoundDeleteView[any])(nil)
	_ views.SetupView      = (*BoundDeleteView[any])(nil)
)

type BoundDeleteView[T any] struct {
	View    *DeleteView[T]
	Context ctx.Context
	Object  T
}

func (l *BoundDeleteView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *BoundDeleteView[T]) Setup(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	return w, r
}

func (l *BoundDeleteView[T]) GetContext(req *http.Request) (c ctx.Context, err error) {
	var ctx = ctx.RequestContext(req)

	if l.View.GetObject != nil {
		obj, err := l.View.GetObject(l, req)
		if err != nil {
			return ctx, err
		}
		l.Object = obj
		ctx.Set("view.object", obj)
	}

	var rt = reflect.TypeFor[T]()
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() == reflect.Struct && rt.NumField() > 0 {
		var field = rt.Field(0)
		var label = field.Tag.Get("label")
		if label == "" {
			var ctype = contenttypes.DefinitionForObject(rt)
			label = strings.ToLower(ctype.Label(req.Context()))
		} else {
			label = strings.ToLower(trans.T(req.Context(), label))
		}

		ctx.Set("view.object.label", label)
	}

	ctx.Set("view.confirm", attrs.ToString(l.Object))

	l.Context = ctx
	if l.View.GetContext != nil {
		c, err = l.View.GetContext(l, ctx)
		if err != nil {
			return ctx, err
		}
	} else {
		c = ctx
	}

	l.Context = c
	return ctx, nil
}

func (l *BoundDeleteView[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) (err error) {
	var next string
	switch v := l.View.NextURL.(type) {
	case func(*BoundDeleteView[T], *http.Request) string:
		next = v(l, req)
	case func(*http.Request) string:
		next = v(req)
	case string:
		next = django.Reverse(v)
	default:
		assert.Fail("submit url not provided for BoundFormModalView")
	}

	context.Set("NextURL", next)

	if req.Method == http.MethodPost {
		if err := req.ParseForm(); err != nil {
			return err
		}

		del := req.PostForm.Get("confirm")
		if del != attrs.ToString(l.Object) {
			messages.Error(req, "Please type the confirmation correctly.")
			http.Redirect(w, req, req.URL.String(), http.StatusFound)
			return nil
		}

		assert.True(l.View.Delete != nil, "view.Delete not set")

		if err = l.View.Delete(l, req, l.Object); err != nil {
			if !errors.Is(err, errors.NotExists) {
				return err
			}
		}

		http.Redirect(w, req, next, http.StatusFound)
		return nil
	}

	var (
		base     = l.View.GetBaseKey()
		template = l.View.GetTemplate(req)
	)

	switch {
	case l.View.Render != nil:
		err = l.View.Render(l, w, req, context)
	case base != "" || template != "":
		err = tpl.FRender(w, context, base, template)
	default:
		return errors.Wrap(errs.ErrNotImplemented, "view.Render not set and no template specified")
	}
	return err
}

type DeleteView[T any] struct {
	BaseKey    string
	Title      any // string | func(*http.Request) string | func(context.Context) string
	Template   any // string | func(*http.Request) string
	NextURL    any // string | func(*http.Request) string | func(view, *http.Request) string
	Render     func(*BoundDeleteView[T], io.Writer, *http.Request, ctx.Context) error
	GetContext func(*BoundDeleteView[T], *ctx.HTTPRequestContext) (ctx.Context, error)
	GetObject  func(*BoundDeleteView[T], *http.Request) (T, error)
	Delete     func(*BoundDeleteView[T], *http.Request, T) error
}

func (l *DeleteView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *DeleteView[T]) Methods() []string {
	return []string{"GET", "POST"}
}

func (l *DeleteView[T]) GetBaseKey() string {
	return l.BaseKey
}

func (l *DeleteView[T]) GetTemplate(req *http.Request) string {
	switch t := l.Template.(type) {
	case string:
		return t
	case func(*http.Request) string:
		return t(req)
	}
	return ""
}

func (l *DeleteView[T]) GetTitle(req *http.Request) string {
	switch t := l.Title.(type) {
	case string:
		return t
	case func(*http.Request) string:
		return t(req)
	case func(context.Context) string:
		return t(req.Context())
	}
	return ""
}

func (l *DeleteView[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	nl := *l
	return &BoundDeleteView[T]{View: &nl}, nil
}
