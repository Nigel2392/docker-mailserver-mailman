package mailmgmt

import (
	"context"
	"fmt"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/chooser"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type chooserFormOptions struct {
	urlParam   string
	urlId      any
	chooserObj attrs.Definer
	chooserKey string
	formName   string
	formLabel  func(ctx context.Context) trans.Translation
	formHelp   func(ctx context.Context) trans.Translation
	formOpts   []func(forms.Form)
}

func newSimpleChooserForm(ctx context.Context, opts chooserFormOptions) *forms.BaseForm {
	var ctype = contenttypes.NewContentType(opts.chooserObj)
	var form = forms.NewBaseForm(ctx)
	var chooserUrl = fmt.Sprintf(
		"%s?%s=%v&exclude=true",
		django.Reverse("chooser:list", ctype.ShortTypeName(), opts.chooserKey),
		opts.urlParam, opts.urlId,
	)

	var chooserAttrs = map[string]string{
		"data-chooser-listurl-value": chooserUrl,
	}

	form.AddField(opts.formName, fields.CharField(
		fields.Required(false),
		fields.Label(opts.formLabel),
		fields.HelpText(opts.formHelp),
		fields.Widget(chooser.NewChooserWidget(
			opts.chooserObj, nil, chooserAttrs, opts.chooserKey,
		)),
	))

	return forms.Initialize(form, opts.formOpts...)
}
