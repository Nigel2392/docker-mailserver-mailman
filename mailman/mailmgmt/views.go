package mailmgmt

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

func (c *MailManagementConfig) ViewIndex(w http.ResponseWriter, r *http.Request) {
	//	if !permissions.HasPermission(r, "mailmgmt.view_addrs") {
	//
	//	}

	var context = ctx.RequestContext(r)

	if err := tpl.FRender(w, context, "main", "mailmgmt/index.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (c *MailManagementConfig) ViewEmails(w http.ResponseWriter, r *http.Request) {
	//	if !permissions.HasPermission(r, "mailmgmt.view_addrs") {
	//
	//	}

	var context = ctx.RequestContext(r)

	emails, err := SetupCtx(r.Context()).Email().List()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	context.Set("emails", emails)

	if err := tpl.FRender(w, context, "main", "mailmgmt/emails.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
