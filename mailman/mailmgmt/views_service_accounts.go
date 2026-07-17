package mailmgmt

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
	"github.com/Nigel2392/mux/middleware/authentication"
)

var ViewServiceAccounts = &list.View[*ServiceAccount]{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "main",
	TemplateName:    "mailmgmt/services/accounts.tmpl",
	PageParam:       "page",
	AmountParam:     "limit",
	OrderableColumns: []string{
		"Identifier",
		"CreatedAt",
		"TokenLastGenerated",
	},
	MaxAmount:     DEFAULT_LIMIT_CHOICES[len(DEFAULT_LIMIT_CHOICES)-1],
	DefaultAmount: DEFAULT_LIMIT_CHOICES[0],
	QuerySet: func(r *http.Request) *queries.QuerySet[*ServiceAccount] {
		var qs = queries.GetQuerySetWithContext(r.Context(), &ServiceAccount{})

		queryValue := r.URL.Query().Get("search")
		if queryValue != "" {
			qs = qs.Filter("Identifier__icontains", queryValue)
		}

		return qs.
			Select("*").
			OrderBy("-CreatedAt")
	},
	GetContextFn: func(r *http.Request, qs *queries.QuerySet[*ServiceAccount]) (ctx.Context, error) {
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
	TitleFieldColumn: func(col list.ListColumn[*ServiceAccount]) list.ListColumn[*ServiceAccount] {
		return list.TitleFieldColumn(col, func(_ *http.Request, _ attrs.Definitions, u *ServiceAccount) string {
			return django.Reverse("mailmgmt:services:detail", u.ID)
		})
	},
	ListColumns: []list.ListColumn[*ServiceAccount]{
		list.Column[*ServiceAccount](
			trans.S("Identifier"),
			"Identifier",
		),
		list.DateTimeFieldColumn[*ServiceAccount](
			trans.LONG_TIME_FORMAT,
			trans.S("Created At"),
			"CreatedAt",
		),
		list.DateTimeFieldColumn[*ServiceAccount](
			trans.LONG_TIME_FORMAT,
			trans.S("Token Last Generated"),
			"TokenLastGenerated",
		),
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *ServiceAccount) template.HTML {

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
				django.Reverse("mailmgmt:services:delete", row.ID), trans.T(r.Context(), "Delete this service account"),
			))
		}),
	},
}

var ViewAddServiceAccount = &views.FormView[*modelforms.BaseModelForm[*ServiceAccount]]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{"GET", "POST"},
		BaseTemplateKey: "main",
		TemplateName:    "mailmgmt/services/add_account.tmpl",
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *modelforms.BaseModelForm[*ServiceAccount]) {
		messages.Success(req, trans.T(req.Context(), "Successfully added service sccount"))
		http.Redirect(w, req, django.Reverse("mailmgmt:services:detail", form.Model.ID), http.StatusSeeOther)
	},
	GetFormFn: func(req *http.Request) *modelforms.BaseModelForm[*ServiceAccount] {
		var f = modelforms.NewBaseModelForm(
			req.Context(), &ServiceAccount{},
		)

		f.SetFields(
			"Identifier",
		)

		f.Load()

		for _, f := range f.Fields() {
			f.SetAttrs(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented small",
			})
		}

		return f
	},
}

var ViewServiceAccountDetail = &views.DetailView[*DetailObject[*ServiceAccount, forms.Form]]{
	URLArgName: "account_id",
	BaseView: views.BaseView{
		BaseTemplateKey: "main",
		TemplateName: []string{
			"mailmgmt/services/detail.tmpl",
		},
		AllowedMethods: []string{"GET", "POST"},
	},
	ChangeContextFn: func(req *http.Request, object *DetailObject[*ServiceAccount, forms.Form], context ctx.ContextWithRequest) ctx.ContextWithRequest {
		context.Set(
			"can_view",
			time.Now().Before(object.Object.TokenLastGenerated.Add(time.Minute*10)),
		)
		context.Set(
			"createdAt", trans.Time(req.Context(), object.Object.CreatedAt, trans.SHORT_TIME_FORMAT),
		)
		context.Set(
			"regeneratedAt", trans.Time(req.Context(), object.Object.TokenLastGenerated, trans.SHORT_TIME_FORMAT),
		)
		return context
	},
	GetObjectFn: func(req *http.Request, urlArg string) (*DetailObject[*ServiceAccount, forms.Form], error) {
		var row, err = queries.
			GetQuerySetWithContext(req.Context(), &ServiceAccount{}).
			Select("*").
			Filter("ID", urlArg).
			Get()

		if err != nil {
			return nil, err
		}

		var form = forms.NewBaseForm(req.Context())
		form.AddField("confirm", fields.CharField(
			fields.Label(trans.S("Confirm")),
			fields.HelpText(trans.S("Please re-type the service account identifier (%q) to confirm", row.Object.Identifier)),
			fields.Attributes(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented small",
			}),
		))

		var detailObj = &DetailObject[*ServiceAccount, forms.Form]{
			Object: row.Object,
			Form:   form,
		}

		return detailObj, nil
	},
	PostMethod: func(d *views.DetailView[*DetailObject[*ServiceAccount, forms.Form]], w http.ResponseWriter, r *http.Request, bound views.View) (http.ResponseWriter, *http.Request) {
		var bv = bound.(*views.BoundDetailView[*DetailObject[*ServiceAccount, forms.Form]])

		var form = forms.Initialize(
			bv.Object.Form,
			forms.WithRequestData(http.MethodPost, r),
		)

		if !forms.IsValid(r.Context(), form) {
			messages.Error(r, trans.T(r.Context(), "Please correctly fill out the confirmation."))
			return w, r
		}

		cleanedData := form.CleanedData()
		confirm, ok := cleanedData["confirm"].(string)
		if !ok || confirm != bv.Object.Object.Identifier {
			form.AddError("confirm",
				errs.Error(trans.T(r.Context(), "The confirmation doesnt match the expected identifier.")))
			return w, r
		}

		bv.Object.Object.SetToken(generateServiceToken())
		err := bv.Object.Object.Update(r.Context())
		if err != nil {
			messages.Error(r, "Could not regenerate service account token.")
			http.Redirect(w, r, django.Reverse("mailmgmt:services:detail", bv.Object.Object.ID), http.StatusFound)
			return nil, nil
		}

		http.Redirect(w, r, django.Reverse("mailmgmt:services:detail", bv.Object.Object.ID), http.StatusFound)
		return nil, nil
	},
}

var ViewDeleteServiceAccount = &DeleteView[*ServiceAccount]{
	BaseKey: "main",
	Template: []string{
		"mailmgmt/base/delete_form.tmpl",
		"mailmgmt/services/delete_account.tmpl",
	},
	NextURL: "mailmgmt:services",
	HasPermission: func(bdv *BoundDeleteView[*ServiceAccount], w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
		row, err := queries.GetQuerySet(&ServiceAccount{}).
			WithContext(r.Context()).
			Select("*").
			Filter("ID", mux.Vars(r).Get("account_id")).
			Get()
		if err != nil {
			messages.Error(r, trans.T(r.Context(), "Error when retrieving user profile"))
			http.Redirect(w, r, django.Reverse("mailmgmt:services"), http.StatusFound)
			return nil, nil
		}

		if !authentication.Retrieve(r).IsAdmin() {
			// return nil, errors.PermissionDenied.Wrap("You cannot delete an administrator.")
			messages.Error(r, trans.T(r.Context(), "You cannot delete service accounts, please ask your administrator!"))
			http.Redirect(w, r, django.Reverse("mailmgmt:services"), http.StatusFound)
			return nil, nil
		}

		bdv.Object = row.Object
		return w, r
	},
	Delete: func(bdv *BoundDeleteView[*ServiceAccount], r *http.Request, la *ServiceAccount) (err error) {
		ctx, tx, err := queries.StartTransaction(r.Context())
		if err != nil {
			return err
		}

		defer tx.Rollback(ctx)

		if err = la.Delete(ctx); err != nil {
			return err
		}

		return tx.Commit(ctx)
	},
}
