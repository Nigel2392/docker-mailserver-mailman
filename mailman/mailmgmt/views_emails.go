package mailmgmt

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func pathCacheKey(request *http.Request, prefix string) string {
	var hash = md5.New()
	hash.Write([]byte(request.URL.String()))
	if prefix != "" {
		return fmt.Sprintf("%s.%x", prefix, hash.Sum(nil))
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

var ViewEmails = &ListView[ListedAddress]{
	RedirectOnMissingQuery: true,
	BaseKey:                "main",
	Template:               "mailmgmt/emails/emails.tmpl",
}

var ViewEmailsHtmx = &ListView[ListedAddress]{
	ReverseURL: "mailmgmt:emails",
	Template:   "mailmgmt/emails/partials/table_list.tmpl",
	GetCount: func(b *BoundListView[ListedAddress], r *http.Request) (int, error) {
		return SetupCtx(r.Context()).Email().CountTotal(b.Query)
	},
	GetObjects: func(b *BoundListView[ListedAddress], r *http.Request, amount, offset int) ([]ListedAddress, error) {
		return SetupCtx(r.Context()).Email().List(&EmailListConfig{
			Page:        b.Page,
			Limit:       b.Limit,
			SearchQuery: b.Query,
		})
	},
}

var ViewAddEmail = &ModalFormView[forms.Form]{
	Template:       "mailmgmt/emails/add.tmpl",
	Title:          trans.S("Add a new E-mail adress"),
	AllowedMethods: []string{"GET", "POST"},
	GetForm: func(r *http.Request) (forms.Form, error) {
		var form = forms.NewBaseForm(
			r.Context(),
			forms.WithFields(
				fields.EmailField(
					fields.Required(true),
					fields.Name("email"),
					fields.Label(trans.S("Email")),
					fields.HelpText(trans.S("Enter the e-mail adress you want to create.")),
					fields.Attributes(map[string]string{
						"autocomplete": "off",
						"class":        "form-control accented",
					}),
				),
				fields.CharField(
					fields.Required(true),
					fields.Name("password"),
					fields.Label(trans.S("Password")),
					fields.HelpText(trans.S("Enter the password for the new e-mail adress.")),
					fields.Widget(widgets.NewPasswordInput(nil)),
					fields.Attributes(map[string]string{
						"autocomplete": "off",
						"class":        "form-control accented",
					}),
				),
				fields.CharField(
					fields.Required(true),
					fields.Name("password_confirm"),
					fields.Label(trans.S("Confirm Password")),
					fields.HelpText(trans.S("Confirm the password for the new e-mail adress.")),
					fields.Widget(widgets.NewPasswordInput(nil)),
					fields.Attributes(map[string]string{
						"autocomplete": "off",
						"class":        "form-control accented",
					}),
				),
			),
		)
		form.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
			var pwd1, ok1 = m["password"]
			var pwd2, ok2 = m["password_confirm"]
			if !ok1 || !ok2 || pwd1 != pwd2 {
				return []error{
					errors.New(trans.T(r.Context(), "Password does not match confirmation.")),
				}
			}
			return nil
		})
		return form, nil
	},
	Validate: func(r *http.Request, f forms.Form) (forms.Form, bool, error) {
		if r.Method != http.MethodPost {
			return f, false, nil
		}

		f = forms.Initialize(f,
			forms.WithRequestData(http.MethodPost, r),
		)

		if !forms.IsValid(r.Context(), f) {
			return f, false, nil
		}

		var (
			c   = f.CleanedData()
			e   = c["email"].(*mail.Address)
			pwd = c["password"].(string)
		)

		if err := SetupCtx(r.Context()).Email().Add(e.Address, pwd); err != nil {
			return f, true, err
		}

		return f, true, nil
	},
}
