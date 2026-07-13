package sieve

import (
	"cmp"
	"context"
	"iter"
	"net/http"
	"slices"
	"strconv"

	"github.com/Nigel2392/docker-mailserver-mailman/mailman/mailmgmt"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

type ForwardedEmailChoice struct {
	Type    string // auth.User | mailmgmt.MailAlias
	Object  attrs.Definer
	Address *drivers.Email
}

var DEFAULT_LIST_LIMIT = 25

func ForwardChoices(ctx context.Context, limitPerQS int, isActive bool) (int, iter.Seq2[*ForwardedEmailChoice, error], error) {
	uQS := queries.GetQuerySetWithContext(ctx, &auth.User{})
	aQS := queries.GetQuerySetWithContext(ctx, &mailmgmt.MailAlias{})

	if isActive {
		uQS = uQS.Filter("IsActive", isActive)
		aQS = aQS.Filter("IsActive", isActive)
	}

	cnt1, userRows, err := uQS.Limit(limitPerQS).IterAll()
	if err != nil {
		return 0, nil, errors.Wrap(err, "UserQuerySet")
	}

	cnt2, aliasRows, err := aQS.Limit(limitPerQS).IterAll()
	if err != nil {
		return 0, nil, errors.Wrap(err, "AliasQuerySet")
	}

	var (
		typ1 = contenttypes.NewContentType(uQS.Meta().Model()).ShortTypeName()
		typ2 = contenttypes.NewContentType(aQS.Meta().Model()).ShortTypeName()
	)

	iterator := func(yield func(*ForwardedEmailChoice, error) bool) {
		for row, err := range userRows {
			if err != nil {
				yield(nil, err)
				return
			}

			var obj = &ForwardedEmailChoice{
				Type:    typ1,
				Address: row.Object.Email,
				Object:  row.Object,
			}
			if !yield(obj, nil) {
				return
			}
		}
		for row, err := range aliasRows {
			if err != nil {
				yield(nil, err)
				return
			}

			var obj = &ForwardedEmailChoice{
				Type:    typ2,
				Address: row.Object.Source,
				Object:  row.Object,
			}
			if !yield(obj, nil) {
				return
			}
		}
	}

	return cnt1 + cnt2, iterator, nil
}

func ViewForwardedEmails(w http.ResponseWriter, r *http.Request) {

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= DEFAULT_LIST_LIMIT {
		limit = DEFAULT_LIST_LIMIT
	}

	cnt1, userRows, err := queries.
		GetQuerySet(&auth.User{}).
		Filter("IsActive", true).
		Limit(limit).
		IterAll()

	except.AssertNil(
		err, http.StatusInternalServerError,
		"Error while querying all users: %v", err,
	)

	cnt2, aliasRows, err := queries.
		GetQuerySet(&mailmgmt.MailAlias{}).
		Filter("IsActive", true).
		Limit(limit).
		IterAll()

	except.AssertNil(
		err, http.StatusInternalServerError,
		"Error while querying all aliasses: %v", err,
	)

	var fwChoices = make([]*ForwardedEmailChoice, 0, cnt1+cnt2)
	for row, err := range userRows {
		except.AssertNil(
			err, http.StatusInternalServerError,
			"Error while querying user row: %v", err,
		)

		fwChoices = append(fwChoices, &ForwardedEmailChoice{
			Type:    "user",
			Object:  row.Object,
			Address: row.Object.Email,
		})
	}

	for row, err := range aliasRows {
		except.AssertNil(
			err, http.StatusInternalServerError,
			"Error while querying user row: %v", err,
		)

		fwChoices = append(fwChoices, &ForwardedEmailChoice{
			Type:    "alias",
			Object:  row.Object,
			Address: row.Object.Source,
		})
	}

	slices.SortFunc(fwChoices, func(a, b *ForwardedEmailChoice) int {
		return cmp.Compare(a.Address.Address, b.Address.Address)
	})

	fwChoices = fwChoices[:limit]

	var context = ctx.RequestContext(r)
	context.Set("forwards", fwChoices)
	if err := tpl.FRender(w, context, "main", "sieve/forwards/forwarded_emails.tmpl"); err != nil {
		except.Fail(http.StatusInternalServerError, "error while rendering template")
	}
}
