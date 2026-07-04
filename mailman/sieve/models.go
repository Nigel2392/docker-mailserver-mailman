package sieve

import (
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/contrib/auth"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type BannedEmail struct {
	ID     drivers.Uint
	Email  *drivers.Email
	Action string
}

func (b *BannedEmail) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Label:   trans.S("ID"),
			Primary: true,
			Null:    false,
			Blank:   false,
		}),
		attrs.NewField(b, "Email", &attrs.FieldConfig{
			Label: trans.S("Email"),
			Null:  false,
			Blank: false,
		}),
		attrs.NewField(b, "Action", &attrs.FieldConfig{
			Label: trans.S("Action"),
			Null:  false,
			Blank: false,
		}),
	).WithTableName("banned_email")
}

type BannedDomain struct {
	ID     drivers.Uint
	Domain string
	Action string
}

func (b *BannedDomain) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Label:   trans.S("ID"),
			Primary: true,
			Null:    false,
			Blank:   false,
		}),
		attrs.NewField(b, "Domain", &attrs.FieldConfig{
			Label:    trans.S("Domain"),
			HelpText: trans.S("e.g., spam.com (do not include the @ symbol)"),
			Null:     false,
			Blank:    false,
		}),
		attrs.NewField(b, "Action", &attrs.FieldConfig{
			Label: trans.S("Action"),
			Null:  false,
			Blank: false,
		}),
	).WithTableName("banned_domain")
}

type ForwardedEmail struct {
	ID          drivers.Uint
	Source      *drivers.Email
	Destination *drivers.Email
	KeepCopy    bool // if true, adds 'keep;' to keep mail in inbox
}

func (b *ForwardedEmail) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Label:   trans.S("ID"),
			Primary: true,
			Null:    false,
			Blank:   false,
		}),
		attrs.NewField(b, "Source", &attrs.FieldConfig{
			Label: trans.S("Source"),
			Null:  false,
			Blank: false,
		}),
		attrs.NewField(b, "Destination", &attrs.FieldConfig{
			Label: trans.S("Destination"),
			Null:  false,
			Blank: false,
		}),
		attrs.NewField(b, "KeepCopy", &attrs.FieldConfig{
			Label: trans.S("Keep Copy"),
		}),
	).WithTableName("forwarded_email")
}

type VacationRule struct {
	ID      drivers.Uint
	For     *auth.User
	Enabled bool
	Days    int // Minimum days between replies to same sender (default 7)
	Subject string
	Message string // Use \n for newlines
}

func (b *VacationRule) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Label:   trans.S("ID"),
			Primary: true,
			Null:    false,
			Blank:   false,
		}),
		attrs.NewField(b, "For", &attrs.FieldConfig{
			Label:       trans.S("For"),
			Null:        false,
			Blank:       false,
			RelOneToOne: attrs.Relate(&auth.User{}, "", nil),
		}),
		attrs.NewField(b, "Enabled", &attrs.FieldConfig{
			Label: trans.S("Enabled"),
		}),
		attrs.NewField(b, "Days", &attrs.FieldConfig{
			Label:    trans.S("Days"),
			HelpText: trans.S("Days until the next vacation reply gets delivered to the same recipient."),
			Default:  7,
			Null:     false,
			Blank:    false,
		}),
		attrs.NewField(b, "Subject", &attrs.FieldConfig{
			Label: trans.S("Email Subject"),
		}),
		attrs.NewField(b, "Message", &attrs.FieldConfig{
			Label: trans.S("Email Message"),
		}),
	).WithTableName("vacation_email_rule")
}
