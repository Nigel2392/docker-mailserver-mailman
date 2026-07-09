package mailmgmt

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type UserMailProfile struct {
	models.Model `table:"mail_quota" label:"User" json:"-"`

	ID      uint64
	Deleted bool // is the user deleted?
	User    *auth.User
	Bytes   uint
}

// FormattedBytes returns a human-readable string for your frontend UI.
func (u *UserMailProfile) FormattedBytes() string {
	if u.Bytes == 0 {
		return "Unlimited"
	}

	b := float64(u.Bytes)

	// Using bitwise shifts (1<<10 = 1024) is the most idiomatic and performant
	// way to handle binary byte thresholds in Go.
	switch {
	case b >= 1<<40: // Terabytes
		return fmt.Sprintf("%.2f TB", b/(1<<40))
	case b >= 1<<30: // Gigabytes
		return fmt.Sprintf("%.2f GB", b/(1<<30))
	case b >= 1<<20: // Megabytes
		return fmt.Sprintf("%.2f MB", b/(1<<20))
	case b >= 1<<10: // Kilobytes
		return fmt.Sprintf("%.2f KB", b/(1<<10))
	default:
		return fmt.Sprintf("%d B", u.Bytes)
	}
}

// DovecotQuota returns the exact string syntax required by the LDAP mailQuota attribute.
func (u *UserMailProfile) DovecotQuota() string {
	if u.Bytes == 0 {
		return "" // Return empty so the LDAP router can skip adding the attribute
	}

	// Dovecot expects limits in Kilobytes by default unless a unit is specified.
	// By explicitly appending 'B', we safely pass the exact raw byte count,
	// preventing any rounding errors on the mail server.
	return fmt.Sprintf("*:storage=%dB", u.Bytes)
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
				attrs.AttrUniqueKey:       true,
				attrs.AttrReverseAliasKey: "Profile",
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
