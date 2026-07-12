package mailmgmt

import (
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/a-h/templ"
)

var (
	_ views.View           = (*ListView[any])(nil)
	_ views.MethodsView    = (*ListView[any])(nil)
	_ views.BindableView   = (*ListView[any])(nil)
	_ views.TemplateKeyer  = (*ListView[any])(nil)
	_ views.TemplateGetter = (*ListView[any])(nil)
	_ views.View           = (*BoundListView[any])(nil)
	_ views.SetupView      = (*BoundListView[any])(nil)
)

func ViewTemplComponent[T views.View](fn func(v T, w io.Writer, r *http.Request, c ctx.Context) templ.Component) func(v T, w io.Writer, r *http.Request, c ctx.Context) error {
	return func(v T, w io.Writer, r *http.Request, c ctx.Context) error {
		var cmp = fn(v, w, r, c)
		return cmp.Render(r.Context(), w)
	}
}

type SetupViewMixin struct {
	Func func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request)
}

func (l SetupViewMixin) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v SetupViewMixin) Setup(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if v.Func != nil {
		return v.Func(w, r)
	}
	return w, r
}

type DjangoListView[T attrs.Definer] struct {
	list.View[T]
	ReverseURL string
}

func (v *DjangoListView[T]) Setup(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if v.ReverseURL != "" {
		var u = *r.URL
		u.Path = django.Reverse(v.ReverseURL)
		w.Header().Set("HX-Push-Url", u.String())
	}
	return w, r
}

type BoundListView[T any] struct {
	View         *ListView[T]
	Page         int
	Limit        int
	Query        string
	LimitChoices []int
	Paginator    pagination.Pagination[T]
}

func (l *BoundListView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *BoundListView[T]) Setup(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	if len(l.View.LimitChoices) > 0 {
		l.LimitChoices = l.View.LimitChoices
	} else {
		l.LimitChoices = DEFAULT_LIMIT_CHOICES
	}

	var shouldRedirect bool
	var q = make(url.Values)
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
		shouldRedirect = true
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || !slices.Contains(l.LimitChoices, limit) {
		limit = l.LimitChoices[0]
		shouldRedirect = true
	}

	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))

	search := r.URL.Query().Get("search")
	if search != "" {
		q.Set("search", search)
	}

	// always be explicit with URLs
	if shouldRedirect && l.View.RedirectOnMissingQuery {
		var newUrl = *r.URL
		newUrl.RawQuery = q.Encode()
		http.Redirect(w, r, newUrl.String(), http.StatusFound)
		return nil, nil
	}

	l.Page = page
	l.Limit = limit
	l.Query = search

	if l.View.GetObjects != nil {
		l.Paginator = &pagination.Paginator[[]T, T]{
			Context: r.Context(),
			Amount:  limit,
			URL:     django.Reverse("mailmgmt:htmx:emails"),
			GetObjects: func(amount, offset int) ([]T, error) {
				return l.View.GetObjects(l, r, amount, offset)
			},
			GetCount: func() (int, error) {
				return l.View.GetCount(l, r)
			},
		}
	}

	if l.View.ReverseURL != "" {
		var u = *r.URL
		u.Path = django.Reverse(l.View.ReverseURL)
		w.Header().Set("HX-Push-Url", u.String())

	}

	return w, r
}

func (l *BoundListView[T]) GetContext(req *http.Request) (c ctx.Context, err error) {
	var ctx = ctx.RequestContext(req)

	ctx.Set("view.page", l.Page)
	ctx.Set("view.limit", l.Limit)
	ctx.Set("view.query", l.Query)
	ctx.Set("view.view", l.View)

	if l.Paginator != nil {
		pageObj, err := l.Paginator.Page(l.Page)
		if err != nil && !errors.Is(err, errors.NoRows) {
			return ctx, err
		}

		pageObj.(interface{ WithAttrs(attrs map[string]any) }).WithAttrs(map[string]any{
			"hx-boost":     "true",
			"hx-target":    "#list-body",
			"hx-swap":      "innerHTML",
			"hx-indicator": "#list-body",
		})

		ctx.Set("view_paginator", l.Paginator)
		ctx.Set("view_paginator_object", pageObj)
	}

	if l.View.LimitChoices != nil {
		ctx.Set("view.limitChoices", l.View.LimitChoices)
	} else {
		ctx.Set("view.limitChoices", DEFAULT_LIMIT_CHOICES)
	}

	if l.View.GetContext != nil {
		c, err = l.View.GetContext(l, ctx)
		if err != nil {
			return ctx, err
		}
	} else {
		c = ctx
	}

	return ctx, nil
}

func (l *BoundListView[T]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) error {
	var (
		base     = l.View.GetBaseKey()
		template = l.View.GetTemplate(req)
	)

	switch {
	case l.View.Render != nil:
		return l.View.Render(l, w, req, context)
	case base != "" || template != "":
		return tpl.FRender(w, context, base, template)
	default:
		return errors.Wrap(errs.ErrNotImplemented, "view.Render not set and no template specified")
	}
}

var DEFAULT_LIMIT_CHOICES = []int{
	10, 25, 50, 100,
}

type ListView[T any] struct {
	ReverseURL             string
	BaseKey                string
	Template               any // string | func(*http.Request) string
	Render                 func(*BoundListView[T], io.Writer, *http.Request, ctx.Context) error
	GetObjects             func(b *BoundListView[T], r *http.Request, amount, offset int) ([]T, error)
	GetCount               func(*BoundListView[T], *http.Request) (int, error)
	GetContext             func(*BoundListView[T], *ctx.HTTPRequestContext) (ctx.Context, error)
	LimitChoices           []int
	RedirectOnMissingQuery bool
}

func (l *ListView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *ListView[T]) Methods() []string {
	return []string{"GET"}
}

func (l *ListView[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	nl := *l
	return &BoundListView[T]{View: &nl}, nil
}

func (l *ListView[T]) GetBaseKey() string {
	return l.BaseKey
}

func (l *ListView[T]) GetTemplate(req *http.Request) string {
	switch t := l.Template.(type) {
	case string:
		return t
	case func(*http.Request) string:
		return t(req)
	}
	return ""
}
