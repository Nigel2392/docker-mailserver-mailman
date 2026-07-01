package mailmgmt

import (
	"fmt"
	"net/http"
	"net/mail"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

func (c *MailManagementConfig) ViewAliases(w http.ResponseWriter, r *http.Request) {
}

func (c *MailManagementConfig) ViewAliasesHtmx(w http.ResponseWriter, r *http.Request) {
}

var ViewAddAliasHtmx = &ModalFormView[forms.Form]{
	Template:       "mailmgmt/emails/modal_form.tmpl",
	SuccessText:    trans.S("Alias created successfully."),
	Title:          trans.S("Add a new E-mail alias"),
	AllowedMethods: []string{"GET", "POST"},
	SubmitURL: func(_ *BoundFormModalView[forms.Form], r *http.Request) string {
		return fmt.Sprintf("%s?email=%s",
			django.Reverse("mailmgmt:htmx:alias:add"),
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
	Validate: func(r *http.Request, f forms.Form) (forms.Form, bool, error) {
		var email = r.URL.Query().Get("email")
		if email == "" {
			return nil, false, errs.ErrFieldRequired
		}

		var addrObj, err = SetupCtx(r.Context()).Email().Get(email)
		if err != nil {
			return nil, false, err
		}

		var (
			c = f.CleanedData()
			a = c["alias"].(*mail.Address)
		)

		if err := SetupCtx(r.Context()).Alias().Add(a.Address, addrObj.Email); err != nil {
			return f, true, err
		}

		return f, true, nil
	},
}
