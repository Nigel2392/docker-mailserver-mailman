package mailmgmt

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/htmx"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

var ViewAliasses = &list.View[*MailAlias]{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "main",
	TemplateName:    "mailmgmt/aliasses/aliasses.tmpl",
	PageParam:       "page",
	AmountParam:     "limit",
	OrderableColumns: []string{
		"Email", "UserCount", "IsActive",
	},
	MaxAmount:     DEFAULT_LIMIT_CHOICES[len(DEFAULT_LIMIT_CHOICES)-1],
	DefaultAmount: DEFAULT_LIMIT_CHOICES[0],
	Mixins: func(r *http.Request, v *list.View[*MailAlias]) []views.View {
		return []views.View{SetupViewMixin{Func: func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
			r = r.WithContext(list.SetAllowListRowSelect(r.Context(), true))
			return w, r
		}}}
	},
	CountQuerySet: func(qs *queries.QuerySet[*MailAlias]) (int64, error) {
		return qs.ClearGroupBy().Count()
	},
	QuerySet: func(r *http.Request) *queries.QuerySet[*MailAlias] {
		qs := queries.GetQuerySetWithContext(r.Context(), &MailAlias{})

		queryValue := r.URL.Query().Get("search")
		if queryValue != "" {
			qs = qs.Filter("Email__icontains", queryValue)
		}

		return qs.
			Select("ID", "Email", "IsActive").
			GroupBy("ID").
			Annotate("UserCount", expr.COUNT("Destination.ID")). // Count the joined user IDs
			OrderBy("-IsActive", "-UserCount", "Email")
	},
	GetContextFn: func(r *http.Request, qs *queries.QuerySet[*MailAlias]) (ctx.Context, error) {
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
	TitleFieldColumn: func(col list.ListColumn[*MailAlias]) list.ListColumn[*MailAlias] {
		//return list.RowSelectColumn(
		//	"list-form",
		//	nil,
		//	nil,
		return list.TitleFieldColumn(col, func(_ *http.Request, _ attrs.Definitions, a *MailAlias) string {
			return django.Reverse("mailmgmt:aliasses:detail", a.ID)
		}) //,
		//	map[string]any{
		//		"data-table-list-target": "selectAll",
		//		"data-action":            "change->table-list#toggleAllCheckboxes",
		//	},
		//	map[string]any{
		//		"data-table-list-target": "checkbox",
		//		"data-action":            "change->table-list#updateSelectAll",
		//	},
		//)
	},
	ListColumns: []list.ListColumn[*MailAlias]{
		list.Column[*MailAlias](
			trans.S("Email"),
			"Email",
		),
		list.FieldColumn[*MailAlias](
			trans.S("User Count"),
			"UserCount",
		),
		list.BooleanFieldColumn[*MailAlias](
			trans.S("IsActive"),
			"IsActive",
		),
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *MailAlias) template.HTML {
			if !authentication.Retrieve(r).IsAdmin() {
				return ""
			}

			var html = `<div class="mailmgmt-list-item-actions">
                <a href="%s" class="mailmgmt-action-button mailmgmt-action-delete">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
                        <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5M11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84l.853-10.66h.538a.5.5 0 0 0 0-1zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5.03a.5.5 0 0 1 .47-.53Zm5.058 0a.5.5 0 0 1 .47.53l-.5 8.5a.5.5 0 1 1-.998-.06l.5-8.5a.5.5 0 0 1 .528-.47M8 4.5a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5"/>
                    </svg>
                </a>
            </div>`

			return template.HTML(fmt.Sprintf(html,
				django.Reverse("mailmgmt:aliasses:delete", row.ID), trans.T(r.Context(), "Delete"),
			))
		}),
	},
}

var ViewAliasDetail = &views.DetailView[*DetailObject[*MailAlias, *forms.BaseForm]]{
	URLArgName: "alias_id",
	BaseView: views.BaseView{
		BaseTemplateKey: "main",
		TemplateName: []string{
			"mailmgmt/base/detail_base.tmpl",
			"mailmgmt/aliasses/detail.tmpl",
		},
		AllowedMethods: []string{"GET", "POST"},
	},
	ChangeContextFn: func(req *http.Request, object *DetailObject[*MailAlias, *forms.BaseForm], context ctx.ContextWithRequest) ctx.ContextWithRequest {
		context.Set("form", object.Form)
		return context
	},
	GetObjectFn: func(req *http.Request, urlArg string) (*DetailObject[*MailAlias, *forms.BaseForm], error) {
		var row, err = queries.
			GetQuerySetWithContext(req.Context(), &MailAlias{}).
			Select("*").
			Preload(queries.Preload{
				Path: "Destination",
				QuerySet: queries.
					GetQuerySet[attrs.Definer](&auth.User{}).
					OrderBy("-IsActive", "Email"),
			}).
			Filter("ID", urlArg).
			Get()
		if err != nil {
			return nil, err
		}

		var obj = &DetailObject[*MailAlias, *forms.BaseForm]{
			Object: row.Object,
			Form: newSimpleChooserForm(req.Context(), chooserFormOptions{
				urlParam:   "alias_id",
				urlId:      row.Object.ID,
				chooserObj: &auth.User{},
				chooserKey: "mailman_user",
				formName:   "user",
				formLabel:  trans.S("User"),
				formHelp:   trans.S("Select a user to assign this alias to."),
				formOpts: []func(forms.Form){
					forms.WithFields(
						fields.BooleanField(
							fields.Name("is_active"),
							fields.Label(trans.S("Is Active")),
							fields.HelpText(trans.S("Wether this user can send and receive e-mails.")),
							fields.Default(row.Object.IsActive),
						),
					),
				},
			}),
		}

		return obj, nil
	},
	PostMethod: func(d *views.DetailView[*DetailObject[*MailAlias, *forms.BaseForm]], w http.ResponseWriter, r *http.Request, bound views.View) (http.ResponseWriter, *http.Request) {
		var bv = bound.(*views.BoundDetailView[*DetailObject[*MailAlias, *forms.BaseForm]])
		var form = forms.Initialize(
			bv.Object.Form,
			forms.WithRequestData(http.MethodPost, r),
		)

		if !forms.IsValid(r.Context(), form) {
			messages.Error(r, trans.T(r.Context(), "Please correctly fill out the form, it is not valid."))
			return w, r
		}

		if !form.HasChanged() {
			logger.Error("The form was not changed, but it was submitted.")
			messages.Error(r, trans.T(r.Context(), "The form was not changed."))
			return w, r
		}

		var ctx, tx, err = queries.StartTransaction(r.Context())
		if err != nil {
			logger.Errorf("failed to start transaction: %v", err)
			messages.Error(r, "Internal server error, no changes were saved.")
			return w, r
		}
		defer tx.Rollback(ctx)

		var (
			cleaned     = form.CleanedData()
			userObj, ok = cleaned["user"]
		)
		if ok && !fields.IsZero(userObj) {
			userRow, err := queries.GetQuerySet(&auth.User{}).
				Filter("ID", userObj).
				Get()
			if err != nil {
				logger.Errorf("Error while retrieving user: %v", err)
				messages.Error(r, "Internal server error, no changes were saved.")
				return w, r
			}

			user := userRow.Object
			qs := bv.Object.Object.Destination.Objects()
			exists, err := qs.Filter("ID", user.ID).Exists()
			if err != nil {
				logger.Errorf("Error while checking if user exists in alias queryset: %v", err)
				messages.Error(r, "Internal server error, no changes were saved.")
				return w, r
			}

			if exists {
				messages.Error(r, fmt.Sprintf(
					"%s is already assigned to %s",
					user.Email.Address,
					bv.Object.Object.Email.Address,
				))
				return w, r
			}

			created, err := qs.AddTarget(user)
			if err != nil {
				logger.Errorf("Error while adding alias to user: %v", err)
				messages.Error(r, "Internal server error, no changes were saved.")
				return w, r
			}

			if !created {
				logger.Errorf(
					"Added alias %q to user %q, but was not created",
					bv.Object.Object.Email.Address,
					user.Email.Address,
				)
			}
		}

		isActive, ok := cleaned["is_active"].(bool)
		if !ok {
			logger.Errorf(
				"Type Mismatch for 'is_active' variable: %T",
				cleaned["is_active"],
			)
		}

		if isActive != bv.Object.Object.IsActive {
			_, err := queries.GetQuerySet(&MailAlias{}).
				ExplicitSave().
				Select("IsActive").
				Filter("ID", bv.Object.Object.ID).
				BulkUpdate(expr.As("IsActive", expr.Value(isActive)))
			if err != nil {
				logger.Errorf(
					"Error while updating active status for alias %q: %v",
					bv.Object.Object.Email.Address, err,
				)
				messages.Error(r, "Internal server error, no changes were saved.")
				return w, r
			}
		}

		if err := tx.Commit(ctx); err != nil {
			logger.Errorf("Failed to save changes to database: %v", err)
			messages.Error(r, "Internal server error, no changes were saved.")
			return w, r
		}

		if htmx.Is(r) {
			return w, r
		}

		messages.Success(r, trans.T(r.Context(), "Updated %q.", bv.Object.Object.Email.Address))
		http.Redirect(w, r, r.URL.Path, http.StatusFound)
		return nil, nil
	},
}

var ViewAliasRemoveUser = &views.DetailView[*MailAlias]{
	URLArgName: "alias_id",
	BaseView: views.BaseView{
		BaseTemplateKey: "main",
		TemplateName: []string{
			"mailmgmt/aliasses/partials/htmx_user_remove.tmpl",
		},
		AllowedMethods: []string{"GET", "POST"},
	},
	GetObjectFn: func(req *http.Request, urlArg string) (*MailAlias, error) {
		var row, err = queries.
			GetQuerySetWithContext(req.Context(), &MailAlias{}).
			Select("*").
			Preload("Destination").
			Filter("ID", urlArg).
			Get()
		return row.Object, err
	},
	PostMethod: func(d *views.DetailView[*MailAlias], w http.ResponseWriter, r *http.Request, bound views.View) (http.ResponseWriter, *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.Errorf("Error while parsing form: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return nil, nil
		}

		var id, err = strconv.Atoi(r.PostForm.Get("confirm"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil, nil
		}

		userRow, err := queries.GetQuerySet(&auth.User{}).
			Filter("ID", id).
			Get()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return w, r
		}

		var (
			bv          = bound.(*views.BoundDetailView[*MailAlias])
			aliasUserQS = bv.Object.Destination.Objects()
		)

		bv.Context.Set("User", userRow.Object)
		bv.Context.Set("Object", bv.Object)

		exists, err := aliasUserQS.Filter("ID", id).Exists()
		if err != nil {
			logger.Errorf("Error while checking if user exists in alias queryset: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return nil, nil
		}

		if !exists {
			//messages.Error(r, fmt.Sprintf(
			//	"Alias is not assigned to %s",
			//	bv.Object.Email.Address,
			//))
			w.WriteHeader(http.StatusBadRequest)
			return nil, nil
		}

		if _, err := aliasUserQS.RemoveTargets(userRow.Object); err != nil {
			logger.Errorf("Error while removing alias from user: %v", err)
			return w, r
		}

		if !htmx.Is(r) {
			http.Redirect(
				w, r,
				django.Reverse("mailmgmt:aliasses:detail", bv.Object.ID),
				http.StatusFound,
			)
			return nil, nil
		}

		if len(bv.Object.Destination.AsList()) == 0 {
			htmx.NewResponse(w).Retarget(".mailmgmt-detail-list")
			fmt.Fprintf(w,
				`<div class="mailmgmt-detail-list"><p class="color-standout">%s</p></div>`,
				trans.T(r.Context(), "No users are using this alias."),
			)
		}

		w.Write([]byte{})
		return nil, nil
	},
}

var ViewAddAliasHtmx = &ModalFormView[forms.Form]{
	GenericModalView: GenericModalView[*BoundFormModalView[forms.Form]]{
		Template:       "mailmgmt/base/modal_form.tmpl",
		Title:          trans.S("Add a new E-mail alias"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SuccessText: trans.S("Alias created successfully."),
	GetForm: func(v *BoundFormModalView[forms.Form], r *http.Request) (forms.Form, error) {
		var form = forms.NewBaseForm(r.Context())
		form.AddField("alias", fields.CharField(
			fields.Required(true),
			fields.Name("alias"),
			fields.Label(trans.S("Alias")),
			fields.Attributes(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented",
			}),
			fields.Widget(NewEmailDomainWidget(nil)),
		))

		return form, nil
	},
	IsValid: func(v *BoundFormModalView[forms.Form], r *http.Request, f forms.Form) (forms.Form, bool, error) {
		var c = f.CleanedData()
		var eml, err = getCleanedEmail(c, "alias")
		if err != nil {
			return nil, false, err
		}

		var ma = &MailAlias{
			Email:    (*drivers.Email)(eml),
			IsActive: true,
		}

		exists, err := queries.
			GetQuerySetWithContext(r.Context(), &MailAlias{}).
			Filter("Email__iexact", ma.Email.Address).
			Exists()
		if err != nil {
			return nil, false, err
		}

		if exists {
			f.AddError("alias", errors.Exists.Wrapf("this alias already exists"))
			return f, false, nil
		}

		ma, err = queries.
			GetQuerySetWithContext(r.Context(), &MailAlias{}).
			Filter("Email__iexact", ma.Email.Address).
			Create(ma)
		if err != nil {
			return nil, false, err
		}

		return f, true, nil
	},
}

var ViewAddAliasToUserHtmx = &ModalFormView[forms.Form]{
	GenericModalView: GenericModalView[*BoundFormModalView[forms.Form]]{
		Template:       "mailmgmt/base/modal_form.tmpl",
		Title:          trans.S("Add a new E-mail alias"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SuccessText: trans.S("Alias created successfully."),
	GetForm: func(v *BoundFormModalView[forms.Form], r *http.Request) (forms.Form, error) {
		var userId = mux.Vars(r).GetInt("email_id")
		except.Assert(
			userId > 0,
			http.StatusBadRequest,
			"invalid user",
		)

		var user, err = queries.
			GetQuerySetWithContext(r.Context(), &auth.User{}).
			Filter("ID", userId).
			Get()
		if err != nil {
			return nil, err
		}

		v.Data["user"] = user.Object

		var form = forms.NewBaseForm(r.Context())
		form.AddField("email", fields.EmailField(
			fields.ReadOnly(true),
			fields.Default(user.Object.Email.Address),
			fields.Name("email"),
			fields.Label(trans.S("Email")),
			fields.Attributes(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented",
			}),
		))
		form.AddField("alias", fields.EmailField(
			fields.Required(true),
			fields.Name("alias"),
			fields.Label(trans.S("Alias")),
			fields.Attributes(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented",
			}),
			fields.Widget(NewEmailDomainWidget(nil)),
		))
		return form, nil
	},
	IsValid: func(v *BoundFormModalView[forms.Form], r *http.Request, f forms.Form) (forms.Form, bool, error) {
		u, ok := v.Data["user"].(*auth.User)
		except.Assert(
			ok, http.StatusInternalServerError,
			"invalid server state",
		)

		var c = f.CleanedData()
		var eml, err = getCleanedEmail(c, "alias")
		if err != nil {
			return nil, false, err
		}

		var ma = &MailAlias{
			Email:    (*drivers.Email)(eml),
			IsActive: true,
		}

		ma, _, err = queries.
			GetQuerySetWithContext(r.Context(), &MailAlias{}).
			Filter("Email__iexact", ma.Email.Address).
			GetOrCreate(ma)
		if err != nil {
			return nil, false, err
		}

		_, err = ma.Destination.Objects().AddTarget(u)
		if err != nil {
			return nil, false, err
		}

		return f, true, nil
	},
}

var ViewDeleteAlias = &DeleteView[*MailAlias]{
	BaseKey: "main",
	Template: []string{
		"mailmgmt/base/delete_form.tmpl",
		"mailmgmt/aliasses/delete_alias.tmpl",
	},
	NextURL: "mailmgmt:aliasses",
	GetObject: func(bdv *BoundDeleteView[*MailAlias], r *http.Request) (*MailAlias, error) {
		row, err := queries.GetQuerySet(&MailAlias{}).
			WithContext(r.Context()).
			Select("*").
			Preload("Destination").
			Filter("ID", mux.Vars(r).Get("alias_id")).
			Get()

		return row.Object, err
	},
	Delete: func(bdv *BoundDeleteView[*MailAlias], r *http.Request, la *MailAlias) (err error) {
		return la.Delete(r.Context())
		//la.IsActive = false
		//return la.Update(r.Context())
	},
}
