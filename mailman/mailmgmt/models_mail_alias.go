package mailmgmt

import (
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	formfields "github.com/Nigel2392/go-django/src/forms/fields"
)

type MailAliasUser struct {
	AliasID drivers.Uint `json:"alias_id" attrs:"primary;readonly"`
	UserID  drivers.Uint `json:"user_id" attrs:"primary;readonly"`
}

func (m *MailAliasUser) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.Unbound("AliasID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "alias_id",
		}),
		attrs.Unbound("UserID", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "user_id",
		}),
	).WithTableName("mail_aliases_users")
}

func (m *MailAliasUser) UniqueTogether() [][]string {
	return [][]string{
		{"AliasID", "UserID"},
	}
}

// MailAlias represents an email forwarding rule in the database
type MailAlias struct {
	models.Model `table:"mail_aliases" json:"-"`

	ID          uint64                                      `json:"id" attrs:"primary;readonly"`
	Source      *drivers.Email                              `json:"source"` // e.g. "info@example.com"
	Destination *queries.RelM2M[*auth.User, *MailAliasUser] `json:"-"`
	IsActive    bool                                        `json:"is_active"`
}

func (u *MailAlias) Fields() []any {
	return []any{
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
			Label:    trans.S("ID"),
			HelpText: trans.S("The unique identifier for this user."),
		}),
		attrs.Unbound("Source", &attrs.FieldConfig{
			Column:    "source",
			FormField: formfields.EmailField,
			Label:     trans.S("Source Email"),
			HelpText:  trans.S("The email address of the alias."),
		}),
		fields.NewManyToManyField[*queries.RelM2M[*auth.User, *MailAliasUser]](
			u, "Destination", &fields.FieldConfig{
				DataModelFieldConfig: fields.DataModelFieldConfig{
					Label:    trans.S("Destination"),
					HelpText: trans.S("Users belonging to this alias."),
				},
				ScanTo:            &u.Destination,
				ReverseName:       "Aliasses",
				NoReverseRelation: false,
				Rel: attrs.Relate(
					&auth.User{}, "",
					&attrs.ThroughModel{
						This:   &MailAliasUser{},
						Source: "AliasID",
						Target: "UserID",
					},
				),
			},
		),
		attrs.Unbound("IsActive", &attrs.FieldConfig{
			Column: "is_active",
			Blank:  true,
		}),
	}
}

func (u *MailAlias) FieldDefs() attrs.Definitions {
	return u.Model.Define(u, u.Fields)
}
