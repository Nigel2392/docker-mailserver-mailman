package mailmgmt

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
)

const CACHE_TIME = time.Second * 300 // 5 minutes

func hashStr(s string) string {
	if s == "" {
		return ""
	}
	var hash = md5.New()
	hash.Write([]byte(s))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

var ViewEmails = &list.View[*auth.User]{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "main",
	TemplateName:    "mailmgmt/emails/emails.tmpl",
	PageParam:       "page",
	AmountParam:     "limit",
	MaxAmount:       DEFAULT_LIMIT_CHOICES[len(DEFAULT_LIMIT_CHOICES)-1],
	DefaultAmount:   DEFAULT_LIMIT_CHOICES[0],
	Mixins: func(r *http.Request, v *list.View[*auth.User]) []views.View {
		return []views.View{SetupViewMixin{Func: func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
			r = r.WithContext(list.SetAllowListRowSelect(r.Context(), true))
			return w, r
		}}}
	},
	GetContextFn: func(r *http.Request, qs *queries.QuerySet[*auth.User]) (ctx.Context, error) {
		c := ctx.RequestContext(r)
		pageValue, _ := strconv.Atoi(r.URL.Query().Get("page"))
		amountValue, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		queryValue := r.URL.Query().Get("search")
		c.Set("view.page", pageValue)
		c.Set("view.limit", amountValue)
		c.Set("view.query", queryValue)
		c.Set("view.limitChoices", DEFAULT_LIMIT_CHOICES)
		return c, nil
	},
	TitleFieldColumn: func(col list.ListColumn[*auth.User]) list.ListColumn[*auth.User] {
		return list.RowSelectColumn(
			"list-form",
			nil,
			nil,
			list.TitleFieldColumn(col, func(_ *http.Request, _ attrs.Definitions, _ *auth.User) string { return "" }),
			map[string]any{
				"data-table-list-target": "selectAll",
				"data-action":            "change->table-list#toggleAllCheckboxes",
			},
			map[string]any{
				"data-table-list-target": "checkbox",
				"data-action":            "change->table-list#updateSelectAll",
			},
		)
	},
	ListColumns: []list.ListColumn[*auth.User]{
		list.Column[*auth.User](
			trans.S("Email"),
			"Email",
		),
		list.FuncColumn(
			trans.S("Name"),
			func(r *http.Request, defs attrs.Definitions, row *auth.User) interface{} {
				return fmt.Sprintf("%s %s", row.FirstName, row.LastName)
			},
		),
		list.BooleanFieldColumn[*auth.User](
			trans.S("IsAdministrator"),
			"IsAdministrator",
		),
		list.BooleanFieldColumn[*auth.User](
			trans.S("IsActive"),
			"IsActive",
		),
		list.DateTimeFieldColumn[*auth.User](
			trans.DEFAULT_TIME_FORMAT,
			trans.S("LastLogin"),
			"LastLogin",
		),
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
					auth.ValidateCharacters(true, auth.ChrFlagAll),
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
	//IsValid: func(r *http.Request, f forms.Form) (forms.Form, bool, error) {
	//	var (
	//		c   = f.CleanedData()
	//		e   = c["email"].(*mail.Address)
	//		pwd = c["password"].(string)
	//	)
	//
	//	if err := SetupCtx(r.Context()).Email().Add(e.Address, pwd); err != nil {
	//		return f, true, err
	//	}
	//
	//	_, err := cache.RollOver(r.Context(), "emails", time.Hour)
	//	return f, true, err
	//},
}

var ViewUpdateEmailPasswordHtmx = &ModalFormView[forms.Form]{
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
					auth.ValidateCharacters(true, auth.ChrFlagAll),
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
	//IsValid: func(r *http.Request, f forms.Form) (forms.Form, bool, error) {
	//	var email = r.URL.Query().Get("email")
	//	if email == "" {
	//		return nil, false, errs.ErrFieldRequired
	//	}
	//
	//	var addrObj, err = SetupCtx(r.Context()).Email().Get(email)
	//	if err != nil {
	//		return nil, false, err
	//	}
	//
	//	var (
	//		c   = f.CleanedData()
	//		pwd = c["password"].(string)
	//	)
	//
	//	if err := SetupCtx(r.Context()).Email().Update(addrObj.Email, pwd); err != nil {
	//		return f, true, err
	//	}
	//
	//	return f, true, err
	//},
}

var ViewDeleteEmail = &DeleteView[*auth.User]{
	BaseKey:  "main",
	Template: "mailmgmt/emails/delete_email.tmpl",
	NextURL:  "mailmgmt:emails",
	GetObject: func(bdv *BoundDeleteView[*auth.User], r *http.Request) (*auth.User, error) {
		var eml, err = mail.ParseAddress(r.URL.Query().Get("email"))
		if err != nil {
			return nil, errs.ErrInvalidSyntax
		}

		row, err := auth.GetUserQuerySet().
			WithContext(r.Context()).
			Filter("email__iexact", eml).
			Get()

		return row.Object, err
	},
	Delete: func(bdv *BoundDeleteView[*auth.User], r *http.Request, la *auth.User) (err error) {
		la.IsActive = false
		return la.Save(r.Context())
	},
}
