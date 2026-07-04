package mailmgmt

import (
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type UserMailQuota struct {
	models.Model `table:"mail_quota" json:"-"`

	ID    uint64
	User  *auth.User
	Bytes uint
}

func (m *UserMailQuota) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.Unbound("ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
			Label:    trans.S("ID"),
		}),
		attrs.Unbound("User", &attrs.FieldConfig{
			ReadOnly:    true,
			RelOneToOne: attrs.Relate(&auth.User{}, "", nil),
			Label:       trans.S("User"),
		}),
		attrs.Unbound("Bytes", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    trans.S("Quota in Bytes"),
		}),
	).WithTableName("user_mail_quota")
}
