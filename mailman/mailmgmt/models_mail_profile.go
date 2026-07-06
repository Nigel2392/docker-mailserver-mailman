package mailmgmt

import (
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
)

type UserMailProfile struct {
	models.Model `table:"mail_quota" json:"-"`

	ID      uint64
	Deleted bool // is the user deleted?
	User    *auth.User
	Bytes   uint
}

func (m *UserMailProfile) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.Unbound("ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
			Label:    trans.S("ID"),
		}),
		attrs.Unbound("User", &attrs.FieldConfig{
			Null:        true,
			ReadOnly:    true,
			RelOneToOne: attrs.Relate(&auth.User{}, "", nil),
			Label:       trans.S("User"),
			Attributes: map[string]interface{}{
				attrs.AttrUniqueKey: true,
			},
		}),
		attrs.Unbound("Deleted", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    trans.S("Deleted"),
		}),
		attrs.Unbound("Bytes", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    trans.S("Quota in Bytes"),
		}),
	).WithTableName("user_mail_profile")
}

type UserMailProfileProxy struct {
	models.Model `table:"mail_quota_proxy" json:"-"`

	*auth.User `proxy:"true"`
	ID         uint64
	Deleted    bool // is the user deleted?
	Bytes      uint
}

func (u *UserMailProfileProxy) UniqueTogether() [][]string {
	return [][]string{}
}

func (u *UserMailProfileProxy) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	return []migrator.Index{}
}

func (m *UserMailProfileProxy) FieldDefs() attrs.Definitions {
	if m.User == nil {
		m.User = &auth.User{}
	}

	return m.Model.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Column:   "id",
			Label:    trans.S("ID"),
			HelpText: trans.S("The unique identifier for this user."),
		}),
		attrs.NewField(m.User, "ID", &attrs.FieldConfig{
			Embedded:     true,
			Primary:      true,
			ReadOnly:     true,
			NameOverride: "UserID",
			Column:       "id",
			Label:        trans.S("ID"),
			HelpText:     trans.S("The unique identifier for this user."),
		}),
		attrs.Unbound("Deleted", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    trans.S("Deleted"),
		}),
		attrs.Unbound("Bytes", &attrs.FieldConfig{
			ReadOnly: true,
			Label:    trans.S("Quota in Bytes"),
		}),
		attrs.NewField(m.User, "Email", &attrs.FieldConfig{
			Embedded:  true,
			Column:    "email",
			MaxLength: 255,
			MinLength: 3,
			FormField: fields.EmailField,
			Label:     trans.S("Email"),
			HelpText:  trans.S("The email address of the user."),
		}),
		attrs.NewField(m.User, "Username", &attrs.FieldConfig{
			Embedded:  true,
			Column:    "username",
			MaxLength: 16,
			MinLength: 3,
			Label:     trans.S("Username"),
			HelpText:  trans.S("The username of the user."),
		}),
		attrs.NewField(m.User, "FirstName", &attrs.FieldConfig{
			Embedded:  true,
			Column:    "first_name",
			MaxLength: 75,
			Label:     trans.S("First Name"),
			HelpText:  trans.S("The first name of the user."),
		}),
		attrs.NewField(m.User, "LastName", &attrs.FieldConfig{
			Embedded:  true,
			Column:    "last_name",
			MaxLength: 75,
			Label:     trans.S("Last Name"),
			HelpText:  trans.S("The last name of the user."),
		}),
		attrs.NewField(m.User, "Password", &attrs.FieldConfig{
			Embedded:  true,
			Column:    "password",
			MaxLength: 255,
			Label:     trans.S("Password"),
			HelpText:  trans.S("The user's password. It is stored as a hash."),
		}),
		attrs.NewField(m.User, "CreatedAt", &attrs.FieldConfig{
			Embedded: true,
			Label:    trans.S("Created At"),
			HelpText: trans.S("The date and time when the user was created."),
			Column:   "created_at",
		}),
		attrs.NewField(m.User, "UpdatedAt", &attrs.FieldConfig{
			Embedded: true,
			Label:    trans.S("Updated At"),
			HelpText: trans.S("The date and time when the user was last updated."),
			Column:   "updated_at",
		}),
		attrs.NewField(m.User, "IsAdministrator", &attrs.FieldConfig{
			Embedded: true,
			Column:   "is_administrator",
			Label:    trans.S("Is Administrator"),
			HelpText: trans.S("Whether the user is an administrator"),
			Blank:    true,
		}),
		attrs.NewField(m.User, "IsActive", &attrs.FieldConfig{
			Embedded: true,
			Column:   "is_active",
			Label:    trans.S("Is Active"),
			HelpText: trans.S("Whether the user is active and can log in."),
			Blank:    true,
			//	Default: django.ConfigGet(
			//		django.Global.Settings,
			//		APPVAR_USER_ACTIVE_DEFAULT,
			//		true, // Default to true if not set
			//	),
		}),
		attrs.NewField(m.User, "LastLogin", &attrs.FieldConfig{
			Embedded: true,
			Column:   "last_login",
			Label:    trans.S("Last Login"),
			HelpText: trans.S("The last time the user logged in."),
			Blank:    true,
			Null:     true,
			ReadOnly: true,
		}),
	)
}
