package sieve

import (
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

type BannedEmail struct {
	Email  *drivers.Email
	Action string
}

func (b *BannedEmail) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
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

type ForwardedEmail struct {
	Source      *drivers.Email
	Destination *drivers.Email
	KeepCopy    bool // if true, adds 'keep;' to keep mail in inbox
}

func (b *ForwardedEmail) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
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
	ForEmail string // The specific email/alias this applies to, e.g., "alice@example.com"
	Enabled  bool
	Days     int // Minimum days between replies to same sender (default 7)
	Subject  string
	Message  string // Use \n for newlines
}

func (b *VacationRule) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, attrs.Field](b,
		attrs.NewField(b, "ForEmail", &attrs.FieldConfig{
			Label: trans.S("ForEmail"),
			Null:  false,
			Blank: false,
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

type SieveConfigData struct {
	BannedEmails []string
	BannedAction string
	Forwards     []ForwardedEmail
	Vacations    []VacationRule
}

const sieveTemplate = `require ["envelope"{{if .Vacations}}, "vacation"{{end}}, "reject"];

{{ if .BannedEmails }}
if address :is "From" [{{ range $i, $e := .BannedEmails }}{{ if (gt $i 0) }}, {{ end }}"{{ $e }}"{{ end }}] {
    {{ if eq .BannedAction "discard" }}discard;{{ else }}reject "Message rejected: Sender is not allowed.";{{ end }}
    stop;
}
{{ end }}

{{range $fwd := $.Forwards}}
if envelope :is "to" "{{$fwd.Source}}" {
    redirect "{{$fwd.Destination}}";
    {{if $fwd.KeepCopy}}keep;{{end}}
}
{{end}}

{{range $vac := $.Vacations}}
{{if $vac.Enabled}}
if envelope :is "to" "{{$vac.ForEmail}}" {
    vacation :days {{$vac.Days}} :subject "{{$vac.Subject}}" "{{$vac.Message}}";
}
{{end}}
{{end}}
`
