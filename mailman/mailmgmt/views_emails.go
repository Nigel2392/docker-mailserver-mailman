package mailmgmt

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/go-signals"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

var ViewEmails = &list.View[*auth.User]{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "main",
	TemplateName:    "mailmgmt/emails/emails.tmpl",
	PageParam:       "page",
	AmountParam:     "limit",
	OrderableColumns: []string{
		"Email",
		"FirstName",
		"Profile.Bytes",
		"IsActive",
	},
	MaxAmount:     DEFAULT_LIMIT_CHOICES[len(DEFAULT_LIMIT_CHOICES)-1],
	DefaultAmount: DEFAULT_LIMIT_CHOICES[0],
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
		list.ChangeColumnType[*auth.User](
			list.FuncColumn(
				trans.S("Name"),
				func(r *http.Request, defs attrs.Definitions, row *auth.User) interface{} {
					if row.FirstName == "" && row.LastName == "" {
						return "---"
					}
					return fmt.Sprintf("%s %s", row.FirstName, row.LastName)
				},
			),
			"FirstName",
		),
		list.ChangeColumnType[*auth.User](
			list.FuncColumn(
				trans.S("Quota"),
				func(r *http.Request, defs attrs.Definitions, row *auth.User) any {
					var profile, ok = defs.Get("Profile").(*UserMailProfile)
					if !ok {
						return ""
					}
					return profile.FormattedBytes()
				},
			),
			"Profile.Bytes",
		),
		list.BooleanFieldColumn[*auth.User](
			trans.S("IsActive"),
			"IsActive",
		),
		//	list.DateTimeFieldColumn[*auth.User](
		//		trans.DEFAULT_TIME_FORMAT,
		//		trans.S("LastLogin"),
		//		"LastLogin",
		//	),
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *auth.User) template.HTML {
			var html = `<div class="mailmgmt-list-item-actions">
		        <button class="mailmgmt-action-button mailmgmt-action-alias"
		            hx-get="%s"
		            hx-target="body"
		            hx-swap="beforeend">
		
		            <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
		                <path d="M2 2a2 2 0 0 0-2 2v8.01A2 2 0 0 0 2 14h5.5a.5.5 0 0 0 0-1H2a1 1 0 0 1-.966-.741l5.64-3.471L8 9.583l7-4.2V8.5a.5.5 0 0 0 1 0V4a2 2 0 0 0-2-2zm3.708 6.208L1 11.105V5.383zM1 4.217V4a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v.217l-7 4.2z"/>
		                <path d="M16 12.5a3.5 3.5 0 1 1-7 0 3.5 3.5 0 0 1 7 0m-3.5-2a.5.5 0 0 0-.5.5v1h-1a.5.5 0 0 0 0 1h1v1a.5.5 0 0 0 1 0v-1h1a.5.5 0 0 0 0-1h-1v-1a.5.5 0 0 0-.5-.5"/>
		            </svg>
		        </button>
		        <a href="%s" class="mailmgmt-action-button mailmgmt-action-delete">
		            <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
		                <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5M11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84l.853-10.66h.538a.5.5 0 0 0 0-1zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5.03a.5.5 0 0 1 .47-.53Zm5.058 0a.5.5 0 0 1 .47.53l-.5 8.5a.5.5 0 1 1-.998-.06l.5-8.5a.5.5 0 0 1 .528-.47M8 4.5a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5"/>
		            </svg>
		        </a>
		    </div>`

			return template.HTML(fmt.Sprintf(html,
				django.Reverse("mailmgmt:htmx:aliasses:add_user", row.ID), trans.T(r.Context(), "Add new alias"),
				// django.Reverse("mailmgmt:htmx:emails:update", row.ID), trans.T(r.Context(), "Change Password"),
				django.Reverse("mailmgmt:emails:delete", row.ID), trans.T(r.Context(), "Delete"),
			))
		}),
	},
}

var _, _ = auth.SignalPreValidateLogonField.Listen(func(s signals.Signal[*auth.FormSignal], fs *auth.FormSignal) error {
	var emailValue, err = getCleanedEmail(fs.CleanedData, "emailDomain")
	if err != nil {
		return errs.NewValidationError("emailDomain", err)
	}

	fs.CleanedData["username"] = emailValue.Address
	fs.CleanedData["email"] = emailValue

	return nil
})

var ViewAddEmailHtmx = &ModalFormView[*auth.BaseUserForm]{
	GenericModalView: GenericModalView[*BoundFormModalView[*auth.BaseUserForm]]{
		Template:       "mailmgmt/base/modal_form.tmpl",
		Title:          trans.S("Add a new E-mail adress"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SuccessText: trans.S("Email created successfully."),
	GetForm: func(v *BoundFormModalView[*auth.BaseUserForm], r *http.Request) (*auth.BaseUserForm, error) {
		var opts = auth.RegisterFormConfig{AskForNames: true, AlwaysAllLoginFields: true}
		var f = auth.UserRegisterForm(r, opts)
		f.DeleteField("email")
		f.DeleteField("username")
		f.AddField("emailDomain", fields.NewField(
			fields.Label(trans.S("Email")),
			fields.HelpText(trans.S("Enter your desired email address before the @ sign.")),
		))
		f.AddWidget("emailDomain", NewEmailDomainWidget(nil))
		f.Ordering([]string{"emailDomain"})

		for head := f.FormFields.Front(); head != nil; head = head.Next() {
			head.Value.SetAttrs(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented",
			})
		}

		return f, nil

	},
	IsValid: func(v *BoundFormModalView[*auth.BaseUserForm], r *http.Request, f *auth.BaseUserForm) (*auth.BaseUserForm, bool, error) {
		_, err := f.Save()
		return f, true, err
	},
}

var ViewUpdateEmailPasswordHtmx = &ModalFormView[forms.Form]{
	GenericModalView: GenericModalView[*BoundFormModalView[forms.Form]]{
		Template:       "mailmgmt/base/modal_form.tmpl",
		Title:          trans.S("Update E-mail password"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SuccessText: trans.S("Email password updated successfully."),
	GetForm: func(v *BoundFormModalView[forms.Form], r *http.Request) (forms.Form, error) {
		var form = forms.NewBaseForm(
			r.Context(), forms.WithFields(
				fields.EmailField(
					fields.Name("email"),
					fields.ReadOnly(true),
					fields.Default(r.URL.Query().Get("email")),
					fields.Label(trans.S("Email")),
				),
				fields.CharField(
					fields.Required(true),
					fields.Name("password"),
					fields.Label(trans.S("Password")),
					fields.HelpText(trans.S("Enter the new password for the e-mail adress.")),
					fields.Widget(widgets.NewPasswordInput(nil)),
					auth.ValidateCharacters(true, auth.ChrFlagAll),
				),
				fields.CharField(
					fields.Required(true),
					fields.Name("password_confirm"),
					fields.Label(trans.S("Confirm Password")),
					fields.HelpText(trans.S("Confirm the new password for the e-mail adress.")),
					fields.Widget(widgets.NewPasswordInput(nil)),
				),
			),
		)
		form.SetValidators(func(f forms.Form, m map[string]interface{}) []error {
			var pwd1, ok1 = m["password"]
			var pwd2, ok2 = m["password_confirm"]
			if !ok1 || !ok2 || pwd1 != pwd2 {
				return []error{
					errors.Wrap(
						autherrors.ErrPasswordInvalid,
						trans.T(r.Context(), "Password does not match confirmation."),
					),
				}
			}
			return nil
		})
		for head := form.FormFields.Front(); head != nil; head = head.Next() {
			head.Value.SetAttrs(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented",
			})
		}

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
	BaseKey: "main",
	Template: []string{
		"mailmgmt/base/delete_form.tmpl",
		"mailmgmt/emails/delete_email.tmpl",
	},
	NextURL: "mailmgmt:emails",
	ExtraMessage: func(bdv *BoundDeleteView[*UserMailProfile], r *http.Request) []string {
		return []string{trans.T(r.Context(), "Aliasses will not be deleted.")}
	},
	HasPermission: func(bdv *BoundDeleteView[*UserMailProfile], w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
		row, err := queries.GetQuerySet(&UserMailProfile{}).
			WithContext(r.Context()).
			Select("*", "User.*").
			Filter("User.ID", mux.Vars(r).Get("email_id")).
			Get()
		if err != nil {
			messages.Error(r, trans.T(r.Context(), "Error when retrieving user profile"))
			http.Redirect(w, r, django.Reverse("mailmgmt:emails"), http.StatusFound)
			return nil, nil
		}

		if row.Object.User.IsAdministrator && !authentication.Retrieve(r).IsAdmin() {
			// return nil, errors.PermissionDenied.Wrap("You cannot delete an administrator.")
			messages.Error(r, trans.T(r.Context(), "You cannot delete an administrator!"))
			http.Redirect(w, r, django.Reverse("mailmgmt:emails"), http.StatusFound)
			return nil, nil
		}

		bdv.Object = row.Object

		return w, r
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
		return la.Delete(r.Context())
	},
}
