package mailmgmt

import (
	"fmt"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

var ViewAliases = &ListView[*MailAlias]{
	RedirectOnMissingQuery: true,
	BaseKey:                "main",
	Template:               "mailmgmt/emails/alias.tmpl",
	GetContext: func(blv *BoundListView[*MailAlias], hc *ctx.HTTPRequestContext) (ctx.Context, error) {
		// messages.Debug(hc.Request(), "Debug message!")
		// messages.Info(hc.Request(), "Info message!")
		// messages.Success(hc.Request(), "Success message!")
		// messages.Warning(hc.Request(), "Warning message!")
		// messages.Error(hc.Request(), "Error message!")
		return hc, nil
	},
}

var ViewAliasesHtmx = &ListView[*MailAlias]{
	ReverseURL: "mailmgmt:aliases",
	Template:   "mailmgmt/aliases/partials/table_list.tmpl",
	GetCount: func(b *BoundListView[*MailAlias], r *http.Request) (int, error) {
		//	return cache.GetItem(
		//		r.Context(),
		//		CACHE_TIME,
		//		-1, "emails", []string{"aliases", "count", hashStr(b.Query)},
		//		func(ctx context.Context) (int, error) {
		//			return SetupCtx(r.Context()).Alias().CountTotal(b.Query)
		//		},
		//	)
		return 0, nil
	},
	GetObjects: func(b *BoundListView[*MailAlias], r *http.Request, amount, offset int) ([]*MailAlias, error) {
		//	return cache.GetItem(
		//		r.Context(),
		//		CACHE_TIME,
		//		-1, "emails", []string{"aliases", "list", strconv.Itoa(b.Page), strconv.Itoa(b.Limit), hashStr(b.Query)},
		//		func(ctx context.Context) ([]*MailAlias, error) {
		//			var l, err = SetupCtx(r.Context()).Alias().List(&AliasListConfig{
		//				Page:        b.Page,
		//				Limit:       b.Limit,
		//				SearchQuery: b.Query,
		//			})
		//			if err != nil {
		//				return nil, err
		//			}
		//			var res = make([]*MailAlias, 0, l.Len())
		//			for k, v := range l.Iterator() {
		//				res = append(res, *MailAlias{
		//					Alias:   k,
		//					Targets: v,
		//				})
		//			}
		//			return res, nil
		//		},
		//	)
		return nil, nil
	},
}

var ViewAddAliasHtmx = &ModalFormView[forms.Form]{
	Template:       "mailmgmt/emails/modal_form.tmpl",
	SuccessText:    trans.S("Alias created successfully."),
	Title:          trans.S("Add a new E-mail alias"),
	AllowedMethods: []string{"GET", "POST"},
	SubmitURL: func(_ *BoundFormModalView[forms.Form], r *http.Request) string {
		return fmt.Sprintf("%s?email=%s",
			django.Reverse("mailmgmt:htmx:aliases:add"),
			r.URL.Query().Get("email"),
		)
	},
	GetForm: func(r *http.Request) (forms.Form, error) {
		var form = forms.NewBaseForm(
			r.Context(), forms.WithFields(
				fields.EmailField(
					fields.ReadOnly(true),
					fields.Default(r.URL.Query().Get("email")),
					fields.Name("email"),
					fields.Label(trans.S("Email")),
					fields.Attributes(map[string]string{
						"autocomplete": "off",
						"class":        "form-control accented",
					}),
				),
				fields.EmailField(
					fields.Required(true),
					fields.Name("alias"),
					fields.Label(trans.S("Alias")),
					fields.Attributes(map[string]string{
						"autocomplete": "off",
						"class":        "form-control accented",
					}),
				),
			),
		)
		return form, nil
	},
	//	IsValid: func(r *http.Request, f forms.Form) (forms.Form, bool, error) {
	//		var email = r.URL.Query().Get("email")
	//		if email == "" {
	//			return nil, false, errs.ErrFieldRequired
	//		}
	//
	//		var addrObj, err = SetupCtx(r.Context()).Email().Get(email)
	//		if err != nil {
	//			return nil, false, err
	//		}
	//
	//		var (
	//			c = f.CleanedData()
	//			a = c["alias"].(*mail.Address)
	//		)
	//
	//		if err := SetupCtx(r.Context()).Alias().Add(a.Address, addrObj.Email); err != nil {
	//			return f, true, err
	//		}
	//
	//		return f, true, nil
	//	},
}
