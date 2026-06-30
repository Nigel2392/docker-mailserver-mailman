package mailmgmt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
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
	_ views.View           = (*BoundFormModalView[forms.Form])(nil)
	_ views.SetupView      = (*BoundFormModalView[forms.Form])(nil)
)

type BoundFormModalView[FORM forms.Form] struct {
	View    *ModalFormView[FORM]
	Context ctx.Context
	Form    FORM
}

func (l *BoundFormModalView[FORM]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

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
	l.Form, valid, err = l.View.Validate(req, l.Form)
	if err != nil {
		return c, err
	}

	ctx.Set("form", l.Form)
	ctx.Set("valid", valid)

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

func (l *BoundFormModalView[FORM]) Render(w http.ResponseWriter, req *http.Request, context ctx.Context) (err error) {
	var (
		base     = l.View.GetBaseKey()
		template = l.View.GetTemplate(req)
		writer   = new(bytes.Buffer)
		uuid     = uuid.New().String()
	)

	context.Set("view.modal", l)
	context.Set("view.modal.id", fmt.Sprintf(
		"modal-%s", uuid,
	))

	switch {
	case l.View.Render != nil:
		err = l.View.Render(l, writer, req, context)
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

func (l *ModalFormView[T]) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (l *ModalFormView[T]) Methods() []string {
	return l.AllowedMethods
}

func (l *ModalFormView[T]) GetBaseKey() string {
	return l.BaseKey
}

func (l *ModalFormView[T]) GetTemplate(req *http.Request) string {
	switch t := l.Template.(type) {
	case string:
		return t
	case func(*http.Request) string:
		return t(req)
	}
	return ""
}

func (l *ModalFormView[T]) GetTitle(req *http.Request) string {
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

type ModalFormView[FORM forms.Form] struct {
	AllowedMethods []string
	BaseKey        string
	Title          any // string | func(*http.Request) string | func(context.Context) string
	Template       any // string | func(*http.Request) string
	Render         func(*BoundFormModalView[FORM], io.Writer, *http.Request, ctx.Context) error
	GetContext     func(*BoundFormModalView[FORM], *ctx.HTTPRequestContext) (ctx.Context, error)
	GetForm        func(r *http.Request) (FORM, error)
	Validate       func(*http.Request, FORM) (f FORM, valid bool, err error)
}

func (l *ModalFormView[T]) Bind(w http.ResponseWriter, req *http.Request) (views.View, error) {
	nl := *l
	var bound *BoundFormModalView[T]
	bound = &BoundFormModalView[T]{
		View: &nl,
	}
	return bound, nil
}
