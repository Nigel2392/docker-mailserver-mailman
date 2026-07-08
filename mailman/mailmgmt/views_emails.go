package mailmgmt

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
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
	QuerySet: func(r *http.Request) *queries.QuerySet[*auth.User] {
		return queries.
			GetQuerySetWithContext(r.Context(), &auth.User{}).
			Select("*", "Profile.*").
			OrderBy("Email")
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
		list.FuncColumn[*auth.User](
			trans.S("Quota"),
			func(r *http.Request, defs attrs.Definitions, row *auth.User) any {
				var profile, ok = defs.Get("Profile").(*UserMailProfile)
				if !ok {
					return ""
				}
				return profile.FormattedBytes()
			},
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
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *auth.User) template.HTML {
			var html = `<div class="mailmgmt-list-item-actions">
                <button class="mailmgmt-action-button mailmgmt-action-alias"
                    hx-get="%s?email=%s"
                    hx-target="body"
                    hx-swap="beforeend">

                    <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
                        <path d="M2 2a2 2 0 0 0-2 2v8.01A2 2 0 0 0 2 14h5.5a.5.5 0 0 0 0-1H2a1 1 0 0 1-.966-.741l5.64-3.471L8 9.583l7-4.2V8.5a.5.5 0 0 0 1 0V4a2 2 0 0 0-2-2zm3.708 6.208L1 11.105V5.383zM1 4.217V4a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v.217l-7 4.2z"/>
                        <path d="M16 12.5a3.5 3.5 0 1 1-7 0 3.5 3.5 0 0 1 7 0m-3.5-2a.5.5 0 0 0-.5.5v1h-1a.5.5 0 0 0 0 1h1v1a.5.5 0 0 0 1 0v-1h1a.5.5 0 0 0 0-1h-1v-1a.5.5 0 0 0-.5-.5"/>
                    </svg>
                </button>
                <button class="mailmgmt-action-button mailmgmt-action-change"
                    hx-get="%s?email=%s"
                    hx-target="body"
                    hx-swap="beforeend">

                    <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
                        <path d="M5.338 1.59a61 61 0 0 0-2.837.856.48.48 0 0 0-.328.39c-.554 4.157.726 7.19 2.253 9.188a10.7 10.7 0 0 0 2.287 2.233c.346.244.652.42.893.533q.18.085.293.118a1 1 0 0 0 .101.025 1 1 0 0 0 .1-.025q.114-.034.294-.118c.24-.113.547-.29.893-.533a10.7 10.7 0 0 0 2.287-2.233c1.527-1.997 2.807-5.031 2.253-9.188a.48.48 0 0 0-.328-.39c-.651-.213-1.75-.56-2.837-.855C9.552 1.29 8.531 1.067 8 1.067c-.53 0-1.552.223-2.662.524zM5.072.56C6.157.265 7.31 0 8 0s1.843.265 2.928.56c1.11.3 2.229.655 2.887.87a1.54 1.54 0 0 1 1.044 1.262c.596 4.477-.787 7.795-2.465 9.99a11.8 11.8 0 0 1-2.517 2.453 7 7 0 0 1-1.048.625c-.28.132-.581.24-.829.24s-.548-.108-.829-.24a7 7 0 0 1-1.048-.625 11.8 11.8 0 0 1-2.517-2.453C1.928 10.487.545 7.169 1.141 2.692A1.54 1.54 0 0 1 2.185 1.43 63 63 0 0 1 5.072.56"/>
                        <path d="M9.5 6.5a1.5 1.5 0 0 1-1 1.415l.385 1.99a.5.5 0 0 1-.491.595h-.788a.5.5 0 0 1-.49-.595l.384-1.99a1.5 1.5 0 1 1 2-1.415"/>
                    </svg>
                </button>
                <a href="%s?email=%s" class="mailmgmt-action-button mailmgmt-action-delete">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
                        <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5M11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84l.853-10.66h.538a.5.5 0 0 0 0-1zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5.03a.5.5 0 0 1 .47-.53Zm5.058 0a.5.5 0 0 1 .47.53l-.5 8.5a.5.5 0 1 1-.998-.06l.5-8.5a.5.5 0 0 1 .528-.47M8 4.5a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5"/>
                    </svg>
                </a>
            </div>`

			var eml = url.QueryEscape(row.Email.Address)
			return template.HTML(fmt.Sprintf(html,
				django.Reverse("mailmgmt:htmx:aliasses:add"), eml, trans.T(r.Context(), "Add new alias"),
				django.Reverse("mailmgmt:htmx:emails:update"), eml, trans.T(r.Context(), "Change Password"),
				django.Reverse("mailmgmt:emails:delete"), eml, trans.T(r.Context(), "Delete"),
			))
		}),
	},
}

var ViewAddEmailHtmx = &ModalFormView[*auth.BaseUserForm]{
	GenericModalView: GenericModalView[*BoundFormModalView[*auth.BaseUserForm]]{
		Template:       "mailmgmt/emails/modal_form.tmpl",
		Title:          trans.S("Add a new E-mail adress"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SubmitURL:   "mailmgmt:htmx:emails:add",
	SuccessText: trans.S("Email created successfully."),
	GetForm: func(r *http.Request) (*auth.BaseUserForm, error) {
		var opts = auth.RegisterFormConfig{AskForNames: true}
		return auth.UserRegisterForm(r, opts), nil
	},
	IsValid: func(r *http.Request, f *auth.BaseUserForm) (*auth.BaseUserForm, bool, error) {
		eml := f.Cleaned["email"].(*drivers.Email)
		emlName := strings.SplitN(eml.Address, "@", 1)
		f.Instance = &auth.User{}
		f.Instance.Username = emlName[0]
		_, err := f.Save()
		return f, true, err
	},
}

var ViewUpdateEmailPasswordHtmx = &ModalFormView[forms.Form]{
	GenericModalView: GenericModalView[*BoundFormModalView[forms.Form]]{
		Template:       "mailmgmt/emails/modal_list.tmpl",
		Title:          trans.S("Update E-mail password"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SuccessText: trans.S("Email password updated successfully."),
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

var ViewDeleteEmail = &DeleteView[*UserMailProfile]{
	BaseKey:  "main",
	Template: "mailmgmt/emails/delete_email.tmpl",
	NextURL:  "mailmgmt:emails",
	GetObject: func(bdv *BoundDeleteView[*UserMailProfile], r *http.Request) (*UserMailProfile, error) {
		var eml, err = mail.ParseAddress(r.URL.Query().Get("email"))
		if err != nil {
			return nil, errs.ErrInvalidSyntax
		}

		row, err := queries.GetQuerySet(&UserMailProfile{}).
			WithContext(r.Context()).
			Select("*", "User.*").
			Filter("User.Email__iexact", eml.Address).
			Get()

		return row.Object, err
	},
	GetContext: func(bdv *BoundDeleteView[*UserMailProfile], hc *ctx.HTTPRequestContext) (ctx.Context, error) {
		var aliassesQs, ok = bdv.Object.User.FieldDefs().Get("Aliasses").(*queries.RelM2M[attrs.Definer, attrs.Definer])
		if !ok {
			panic(fmt.Sprintf("could not convert %T", bdv.Object.User.FieldDefs().Get("Aliasses")))
		}

		var aliasRows, err = aliassesQs.Objects().All()
		if err != nil {
			return nil, err
		}

		hc.Set("aliasses", aliasRows)
		return hc, nil
	},
	Delete: func(bdv *BoundDeleteView[*UserMailProfile], r *http.Request, la *UserMailProfile) (err error) {
		la.Deleted = true
		return la.Save(r.Context())
	},
}
