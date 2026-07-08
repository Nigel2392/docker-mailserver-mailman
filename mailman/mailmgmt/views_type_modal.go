package mailmgmt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	_ views.View           = (*ModalFormView[forms.Form])(nil)
	_ views.MethodsView    = (*ModalFormView[forms.Form])(nil)
	_ views.BindableView   = (*ModalFormView[forms.Form])(nil)
	_ views.TemplateKeyer  = (*ModalFormView[forms.Form])(nil)
	_ views.TemplateGetter = (*ModalFormView[forms.Form])(nil)
	_ views.Renderer       = (*BoundFormModalView[forms.Form])(nil)
	_ views.View           = (*BoundFormModalView[forms.Form])(nil)
	_ views.SetupView      = (*BoundFormModalView[forms.Form])(nil)
)

type baseView[BOUND any] interface {
	views.View
	views.TemplateKeyer
	views.TemplateGetter
	GetTitle(req *http.Request) string
	RenderFunc() func(BOUND, io.Writer, *http.Request, ctx.Context) error
}

type GenericBoundModalView[_BOUND any, VIEW baseView[_BOUND]] struct {
	Embedder _BOUND
	View     VIEW
}

func (l *GenericBoundModalView[_BOUND, VIEW]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *GenericBoundModalView[_BOUND, VIEW]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) (err error) {
	var (
		base       = l.View.GetBaseKey()
		template   = l.View.GetTemplate(req)
		renderFunc = l.View.RenderFunc()
		writer     = new(bytes.Buffer)
		uuid       = uuid.New().String()
	)

	context.Set("view.modal", l)
	context.Set("view.modal.id", fmt.Sprintf(
		"modal-%s", uuid,
	))

	switch {
	case renderFunc != nil:
		err = renderFunc(l.Embedder, writer, req, context)
	case base != "" || template != "":
		err = tpl.FRender(writer, context, base, template)
	default:
		return errors.Wrap(errs.ErrNotImplemented, "view.Render not set and no template specified")
	}
	if err != nil {
		return err
	}

	var modalTitle string
	if title := l.View.GetTitle(req); title != "" {
		modalTitle = fmt.Sprintf(`<div class="modal-header">%s</div>`, title)
	}

	fmt.Fprintf(w, `<div id="modal-%s" data-controller="form-modal">
    	<div class="modal-underlay" data-form-modal-target="underlay" data-action="click->form-modal#close"></div>
    	<div class="modal-container" data-form-modal-target="modal">
    		%s
    		<div class="modal-content">%s</div>
    		<button class="button modal-close-button" data-action="click->form-modal#close">
				<svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon bg-danger" viewBox="0 0 16 16">
				  	<path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708"/>
				</svg>
			</button>
    	</div>
    </div>`,
		uuid,
		modalTitle,
		writer.String(),
		// trans.T(req.Context(), "Close"),
	)

	return nil
}

type BoundModalView[T baseView[*BoundModalView[T]]] struct {
	GenericBoundModalView[*BoundModalView[T], T]
}

type BoundFormModalView[FORM forms.Form] struct {
	GenericBoundModalView[*BoundFormModalView[FORM], *ModalFormView[FORM]]
	Context ctx.Context
	Form    FORM
}

func (l *BoundFormModalView[FORM]) Setup(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
	assert.True(
		l.View.GetForm != nil,
		"View %T must provide GetForm", l.View,
	)

	var err error
	l.Form, err = l.View.GetForm(r)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			err,
		)
	}

	return w, r
}

func (l *BoundFormModalView[FORM]) GetContext(req *http.Request) (c ctx.Context, err error) {
	var ctx = ctx.RequestContext(req)

	var valid bool
	if req.Method == http.MethodPost {

		l.Form = forms.Initialize(l.Form,
			forms.WithRequestData(http.MethodPost, req),
		)

		if !forms.IsValid(req.Context(), l.Form) {
			valid = false
			goto setupCtx
		}

		l.Form, valid, err = l.View.IsValid(req, l.Form)
		if err != nil {
			return c, err
		}
	}

setupCtx:
	ctx.Set("form", l.Form)
	ctx.Set("valid", valid)
	ctx.Set("success_text", trans.GetTextFunc(l.View.SuccessText)(req.Context()))
	switch v := l.View.SubmitURL.(type) {
	case func(*BoundFormModalView[FORM], *http.Request) string:
		ctx.Set("submit_url", v(l, req))
	case func(*http.Request) string:
		ctx.Set("submit_url", v(req))
	case string:
		ctx.Set("submit_url", django.Reverse(v))
	default:
		assert.Fail("submit url not provided for BoundFormModalView")
	}

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

type GenericModalView[_BOUND views.View] struct {
	AllowedMethods []string
	BaseKey        string
	Title          any // string | func(*http.Request) string | func(context.Context) string
	Template       any // string | func(*http.Request) string
	Render         func(_BOUND, io.Writer, *http.Request, ctx.Context) error
	GetContext     func(_BOUND, *ctx.HTTPRequestContext) (ctx.Context, error)
}

func (l *GenericModalView[_BOUND]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *GenericModalView[_BOUND]) Methods() []string {
	return l.AllowedMethods
}

func (l *GenericModalView[_BOUND]) RenderFunc() func(_BOUND, io.Writer, *http.Request, ctx.Context) error {
	return l.Render
}

func (l *GenericModalView[_BOUND]) GetBaseKey() string {
	return l.BaseKey
}

func (l *GenericModalView[_BOUND]) GetTemplate(req *http.Request) string {
	switch t := l.Template.(type) {
	case string:
		return t
	case func(*http.Request) string:
		return t(req)
	}
	return ""
}

func (l *GenericModalView[_BOUND]) GetTitle(req *http.Request) string {
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

func (l *GenericModalView[_BOUND]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	var bound = &GenericBoundModalView[_BOUND, *GenericModalView[_BOUND]]{}
	bound.View = l
	bound.Embedder = (any(l)).(_BOUND)
	return bound, nil
}

type ModalView struct {
	GenericModalView[*BoundModalView[*ModalView]]
}

type ModalFormView[FORM forms.Form] struct {
	GenericModalView[*BoundFormModalView[FORM]]
	SubmitURL   any // string | func(*http.Request) string | func(view, *http.Request) string
	SuccessText any // trans.GetText
	GetForm     func(r *http.Request) (FORM, error)
	IsValid     func(*http.Request, FORM) (f FORM, valid bool, err error)
}

func (l *ModalFormView[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	nl := *l
	var bound = &BoundFormModalView[T]{}
	bound.GenericBoundModalView.View = &nl
	bound.GenericBoundModalView.Embedder = bound
	return bound, nil
}
