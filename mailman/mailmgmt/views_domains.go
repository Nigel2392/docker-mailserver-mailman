package mailmgmt

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/messages"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

var ViewDomains = &list.View[*Domain]{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "main",
	TemplateName:    "mailmgmt/domains/domains.tmpl",
	PageParam:       "page",
	AmountParam:     "limit",
	MaxAmount:       DEFAULT_LIMIT_CHOICES[len(DEFAULT_LIMIT_CHOICES)-1],
	DefaultAmount:   DEFAULT_LIMIT_CHOICES[0],
	Mixins: func(r *http.Request, v *list.View[*Domain]) []views.View {
		return []views.View{SetupViewMixin{Func: func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
			r = r.WithContext(list.SetAllowListRowSelect(r.Context(), true))
			return w, r
		}}}
	},
	GetContextFn: func(r *http.Request, qs *queries.QuerySet[*Domain]) (ctx.Context, error) {
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
	TitleFieldColumn: func(col list.ListColumn[*Domain]) list.ListColumn[*Domain] {
		return list.RowSelectColumn(
			"list-form",
			nil,
			nil,
			list.TitleFieldColumn(col, func(_ *http.Request, _ attrs.Definitions, _ *Domain) string { return "" }),
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
	ListColumns: []list.ListColumn[*Domain]{
		list.Column[*Domain](
			trans.S("Name"),
			"Name",
		),
		list.Column[*Domain](
			trans.S("Domain"),
			"Domain",
		),
		list.BooleanFieldColumn[*Domain](
			trans.S("IsActive"),
			"IsActive",
		),
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *Domain) template.HTML {
			var html = `<div class="mailmgmt-list-item-actions">
                <a href="%s" class="mailmgmt-action-button mailmgmt-action-delete">
                    <svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" class="mailmgmt-action-icon" viewBox="0 0 16 16" data-controller="tooltip" data-tooltip-content-value="%s" data-tooltip-placement-value="bottom">
                        <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5M11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H1.5a.5.5 0 0 0 0 1h.538l.853 10.66A2 2 0 0 0 4.885 16h6.23a2 2 0 0 0 1.994-1.84l.853-10.66h.538a.5.5 0 0 0 0-1zm1.958 1-.846 10.58a1 1 0 0 1-.997.92h-6.23a1 1 0 0 1-.997-.92L3.042 3.5zm-7.487 1a.5.5 0 0 1 .528.47l.5 8.5a.5.5 0 0 1-.998.06L5 5.03a.5.5 0 0 1 .47-.53Zm5.058 0a.5.5 0 0 1 .47.53l-.5 8.5a.5.5 0 1 1-.998-.06l.5-8.5a.5.5 0 0 1 .528-.47M8 4.5a.5.5 0 0 1 .5.5v8.5a.5.5 0 0 1-1 0V5a.5.5 0 0 1 .5-.5"/>
                    </svg>
                </a>
            </div>`

			return template.HTML(fmt.Sprintf(html,
				django.Reverse("mailmgmt:domains:delete", row.ID), trans.T(r.Context(), "Delete"),
			))
		}),
	},
}

var ViewAddDomain = &views.FormView[*modelforms.BaseModelForm[*Domain]]{
	BaseView: views.BaseView{
		AllowedMethods:  []string{"GET", "POST"},
		BaseTemplateKey: "main",
		TemplateName:    "mailmgmt/domains/add_domain.tmpl",
	},
	SuccessFn: func(w http.ResponseWriter, req *http.Request, form *modelforms.BaseModelForm[*Domain]) {
		messages.Success(req, trans.T(req.Context(), "Successfully added domain"))
		http.Redirect(w, req, django.Reverse("mailmgmt:domains"), http.StatusSeeOther)
	},
	GetFormFn: func(req *http.Request) *modelforms.BaseModelForm[*Domain] {
		var f = modelforms.NewBaseModelForm(
			req.Context(), &Domain{},
		)

		f.SetFields(
			"Name",
			"Domain",
			"IsActive",
		)

		f.Load()

		return f
	},
}

var ViewDeleteDomain = &DeleteView[*Domain]{
	BaseKey:  "main",
	Template: "mailmgmt/domains/delete_domain.tmpl",
	NextURL:  "mailmgmt:domains",
	GetObject: func(bdv *BoundDeleteView[*Domain], r *http.Request) (*Domain, error) {
		row, err := queries.GetQuerySet(&Domain{}).
			WithContext(r.Context()).
			Filter("ID", mux.Vars(r).Get("domain_id")).
			Get()

		return row.Object, err
	},
	ExtraMessage: func(bdv *BoundDeleteView[*Domain], r *http.Request) []string {
		return []string{
			trans.T(r.Context(), "It will also set all (below) listed aliasses and users to inactive."),
		}
	},
	GetContext: func(bdv *BoundDeleteView[*Domain], hc *ctx.HTTPRequestContext) (ctx.Context, error) {
		var domainEmailPart = fmt.Sprintf("@%s", bdv.Object.Domain)
		users, err := queries.GetQuerySet(&auth.User{}).
			Filter("Email__iendswith", domainEmailPart).
			Filter("IsActive", true).
			All()
		if err != nil {
			return nil, err
		}

		aliasses, err := queries.GetQuerySet(&MailAlias{}).
			Filter("Source__iendswith", domainEmailPart).
			Filter("IsActive", true).
			All()
		if err != nil {
			return nil, err
		}

		hc.Set("users", users)
		hc.Set("aliasses", aliasses)

		return hc, nil
	},
	Delete: func(bdv *BoundDeleteView[*Domain], r *http.Request, la *Domain) (err error) {
		var domainEmailPart = fmt.Sprintf("@%s", bdv.Object.Domain)
		qs := queries.GetQuerySet(&auth.User{})
		tx, err := qs.StartTransaction(r.Context())
		if err != nil {
			return err
		}

		defer tx.Rollback(qs.Context())

		_, err = queries.GetQuerySetWithContext(qs.Context(), &auth.User{}).
			Select("IsActive").
			Filter("Email__iendswith", domainEmailPart).
			Filter("IsActive", true).
			BulkUpdate(map[string]expr.Expression{
				"IsActive": expr.V(false),
			})
		if err != nil {
			return err
		}

		_, err = queries.GetQuerySetWithContext(qs.Context(), &MailAlias{}).
			Select("IsActive").
			Filter("Source__iendswith", domainEmailPart).
			Filter("IsActive", true).
			BulkUpdate(map[string]expr.Expression{
				"IsActive": expr.V(false),
			})
		if err != nil {
			return err
		}

		if err = la.Delete(qs.Context()); err != nil {
			return err
		}

		return tx.Commit(qs.Context())
	},
}

var ViewDeactivateDomain = &DeleteView[*Domain]{
	BaseKey:  "main",
	Template: "mailmgmt/domains/delete_domain.tmpl",
	NextURL:  "mailmgmt:domains",
	GetObject: func(bdv *BoundDeleteView[*Domain], r *http.Request) (*Domain, error) {
		row, err := queries.GetQuerySet(&Domain{}).
			WithContext(r.Context()).
			Filter("ID", mux.Vars(r).Get("domain_id")).
			Get()

		return row.Object, err
	},
	ExtraMessage: func(bdv *BoundDeleteView[*Domain], r *http.Request) []string {
		return []string{
			trans.T(r.Context(), "It will also set all (below) listed aliasses and users to inactive."),
		}
	},
	GetContext: func(bdv *BoundDeleteView[*Domain], hc *ctx.HTTPRequestContext) (ctx.Context, error) {
		var domainEmailPart = fmt.Sprintf("@%s", bdv.Object.Domain)
		users, err := queries.GetQuerySet(&auth.User{}).
			Filter("Email__iendswith", domainEmailPart).
			Filter("IsActive", true).
			All()
		if err != nil {
			return nil, err
		}

		aliasses, err := queries.GetQuerySet(&MailAlias{}).
			Filter("Source__iendswith", domainEmailPart).
			Filter("IsActive", true).
			All()
		if err != nil {
			return nil, err
		}

		hc.Set("users", users)
		hc.Set("aliasses", aliasses)

		return hc, nil
	},
	Delete: func(bdv *BoundDeleteView[*Domain], r *http.Request, la *Domain) (err error) {
		var domainEmailPart = fmt.Sprintf("@%s", bdv.Object.Domain)
		qs := queries.GetQuerySet(&auth.User{})
		tx, err := qs.StartTransaction(r.Context())
		if err != nil {
			return err
		}
		defer tx.Rollback(qs.Context())

		_, err = queries.GetQuerySetWithContext(qs.Context(), &auth.User{}).
			Select("IsActive").
			Filter("Email__iendswith", domainEmailPart).
			Filter("IsActive", true).
			BulkUpdate(&auth.User{Base: users.Base{IsActive: false}})
		if err != nil {
			return err
		}

		_, err = queries.GetQuerySetWithContext(qs.Context(), &MailAlias{}).
			Select("IsActive").
			Filter("Source__iendswith", domainEmailPart).
			Filter("IsActive", true).
			BulkUpdate(&MailAlias{IsActive: false})
		if err != nil {
			return err
		}

		la.IsActive = false
		err = la.Update(qs.Context())
		if err != nil {
			return err
		}

		return tx.Commit(qs.Context())
	},
}
