package mailmgmt

import (
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

func (c *MailManagementConfig) ViewIndex(w http.ResponseWriter, r *http.Request) {
	//	if !permissions.HasPermission(r, "mailmgmt.view_addrs") {
	//
	//	}

	context := ctx.RequestContext(r)
	userCount, err := queries.GetQuerySet(&auth.User{}).Count()
	if err != nil {
		except.Fail(http.StatusInternalServerError, err)
	}

	aliasCount, err := queries.GetQuerySet(&MailAlias{}).Count()
	if err != nil {
		except.Fail(http.StatusInternalServerError, err)
	}

	domainCount, err := queries.GetQuerySet(&Domain{}).Count()
	if err != nil {
		except.Fail(http.StatusInternalServerError, err)
	}

	context.Set("userCount", userCount)
	context.Set("aliasCount", aliasCount)
	context.Set("domainCount", domainCount)

	if err := tpl.FRender(w, context, "main", "mailmgmt/index.tmpl"); err != nil {
		except.Fail(http.StatusInternalServerError, err)
	}
}
