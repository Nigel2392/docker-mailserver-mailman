package mailmgmt

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"net/mail"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type emailAddrObjectKey struct{}

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

var ViewDeleteEmail = &DeleteView[*ListedAddress]{
	BaseKey:  "main",
	Template: "mailmgmt/emails/delete_email.tmpl",
	NextURL:  "mailmgmt:emails",
	GetObject: func(bdv *BoundDeleteView[*ListedAddress], r *http.Request) (*ListedAddress, error) {
		var eml, err = mail.ParseAddress(r.URL.Query().Get("email"))
		if err != nil {
			return nil, errs.ErrInvalidSyntax
		}

		return SetupCtx(r.Context()).Email().Get(eml.Address)
	},
	Delete: func(bdv *BoundDeleteView[*ListedAddress], r *http.Request, la *ListedAddress) error {
		return SetupCtx(r.Context()).Email().Delete(la.Email)
	},
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

var ViewAddEmailHtmx = &ModalFormView[forms.Form]{
	Template:       "mailmgmt/emails/modal_form.tmpl",
	SubmitURL:      "mailmgmt:htmx:emails:add",
	SuccessText:    trans.S("Email created successfully."),
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

var ViewUpdateEmailHtmx = &ModalFormView[forms.Form]{
	Template:       "mailmgmt/emails/modal_form.tmpl",
	SuccessText:    trans.S("Email password updated successfully."),
	Title:          trans.S("Update E-mail password"),
	AllowedMethods: []string{"GET", "POST"},
	SubmitURL: func(_ *BoundFormModalView[forms.Form], r *http.Request) string {
		return fmt.Sprintf("%s?email=%s",
			django.Reverse("mailmgmt:htmx:emails:update"),
			r.URL.Query().Get("email"),
		)
	},
	GetForm: func(r *http.Request) (forms.Form, error) {

		var form = forms.NewBaseForm(
			r.Context(), forms.WithFields(
				fields.EmailField(
					fields.Name("email"),
					fields.ReadOnly(true),
					fields.Default(r.URL.Query().Get("email")),
					fields.Label(trans.S("Email")),
					fields.Attributes(map[string]string{
						"autocomplete": "off",
						"class":        "form-control accented",
					}),
				),
				fields.CharField(
					fields.Required(true),
					fields.Name("password"),
					fields.Label(trans.S("Password")),
					fields.HelpText(trans.S("Enter the new password for the e-mail adress.")),
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
					fields.HelpText(trans.S("Confirm the new password for the e-mail adress.")),
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
		var email = r.URL.Query().Get("email")
		if email == "" {
			return nil, false, errs.ErrFieldRequired
		}

		var addrObj, err = SetupCtx(r.Context()).Email().Get(email)
		if err != nil {
			return nil, false, err
		}

		var (
			c   = f.CleanedData()
			pwd = c["password"].(string)
		)

		if err := SetupCtx(r.Context()).Email().Update(addrObj.Email, pwd); err != nil {
			return f, true, err
		}

		return f, true, nil
	},
}
