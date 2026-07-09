package mailmgmt

import (
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/go-django/src/views/list"
	"github.com/Nigel2392/mux"
)

var ViewAliasses = &list.View[*MailAlias]{
	AllowedMethods:  []string{http.MethodGet},
	BaseTemplateKey: "main",
	TemplateName:    "mailmgmt/aliasses/aliasses.tmpl",
	PageParam:       "page",
	AmountParam:     "limit",
	MaxAmount:       DEFAULT_LIMIT_CHOICES[len(DEFAULT_LIMIT_CHOICES)-1],
	DefaultAmount:   DEFAULT_LIMIT_CHOICES[0],
	Mixins: func(r *http.Request, v *list.View[*MailAlias]) []views.View {
		return []views.View{SetupViewMixin{Func: func(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request) {
			r = r.WithContext(list.SetAllowListRowSelect(r.Context(), true))
			return w, r
		}}}
	},
	QuerySet: func(r *http.Request) *queries.QuerySet[*MailAlias] {
		return queries.
			GetQuerySetWithContext(r.Context(), &MailAlias{}).
			Select("ID", "Source", "IsActive").
			GroupBy("ID").
			Annotate("UserCount", expr.COUNT("Destination.ID")). // Count the joined user IDs
			OrderBy("Source")
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
		return list.RowSelectColumn(
			"list-form",
			nil,
			nil,
			list.TitleFieldColumn(col, func(_ *http.Request, _ attrs.Definitions, _ *MailAlias) string { return "" }),
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
	ListColumns: []list.ListColumn[*MailAlias]{
		list.Column[*MailAlias](
			trans.S("Email"),
			"Source",
		),
		list.FuncColumn(
			trans.S("User Count"),
			func(r *http.Request, defs attrs.Definitions, row *MailAlias) interface{} {
				return row.Annotations["UserCount"]
			},
		),
		list.BooleanFieldColumn[*MailAlias](
			trans.S("IsActive"),
			"IsActive",
		),
		list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *MailAlias) template.HTML {
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

var ViewAddAliasHtmx = &ModalFormView[forms.Form]{
	GenericModalView: GenericModalView[*BoundFormModalView[forms.Form]]{
		Template:       "mailmgmt/base/modal_form.tmpl",
		Title:          trans.S("Add a new E-mail alias"),
		AllowedMethods: []string{"GET", "POST"},
	},
	SuccessText: trans.S("Alias created successfully."),
	GetForm: func(v *BoundFormModalView[forms.Form], r *http.Request) (forms.Form, error) {
		var form = forms.NewBaseForm(r.Context())
		form.AddField("alias", fields.EmailField(
			fields.Required(true),
			fields.Name("alias"),
			fields.Label(trans.S("Alias")),
			fields.Attributes(map[string]string{
				"autocomplete": "off",
				"class":        "form-control accented",
			}),
		))

		return form, nil
	},
	IsValid: func(v *BoundFormModalView[forms.Form], r *http.Request, f forms.Form) (forms.Form, bool, error) {
		var c = f.CleanedData()
		var ma = &MailAlias{
			Source:   (*drivers.Email)(c["alias"].(*mail.Address)),
			IsActive: true,
		}

		exists, err := queries.
			GetQuerySetWithContext(r.Context(), &MailAlias{}).
			Filter("Source__iexact", ma.Source.Address).
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
			Filter("Source__iexact", ma.Source.Address).
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
		var ma = &MailAlias{
			Source:   (*drivers.Email)(c["alias"].(*mail.Address)),
			IsActive: true,
		}

		ma, _, err := queries.
			GetQuerySetWithContext(r.Context(), &MailAlias{}).
			Filter("Source__iexact", ma.Source.Address).
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
	BaseKey:  "main",
	Template: "mailmgmt/aliasses/delete_alias.tmpl",
	NextURL:  "mailmgmt:aliasses",
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
