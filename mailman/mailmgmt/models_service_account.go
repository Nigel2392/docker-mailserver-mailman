package mailmgmt

import (
	"context"
	"fmt"
	"hash/fnv"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/google/uuid"
)

//	return queries.GetQuerySet(&Domain{}).
//		WithContext(ctx).
//		Filter("Domain__iexact", domain)

func generateServiceToken() string {
	return uuid.New().String()
}

type ServiceAccount struct {
	models.Model `table:"service_accounts" label:"Service Account"`
	ID           int64

	Identifier         string    // LDAP uid
	Token              string    // LDAP Token
	CreatedAt          time.Time // when it was created
	TokenLastGenerated time.Time
}

func (n *ServiceAccount) String() string {
	return n.Identifier
}

func (o *ServiceAccount) BeforeCreate(ctx context.Context) error {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now()
	}
	if o.Token == "" {
		o.SetToken(generateServiceToken())
	}
	return nil
}

func (o *ServiceAccount) SetToken(token string) {
	var tokenHash = fnv.New128()
	tokenHash.Write([]byte(token))
	o.Token = fmt.Sprintf("%x", tokenHash.Sum(nil))
	o.TokenLastGenerated = time.Now()
}

func (n *ServiceAccount) FieldDefs() attrs.Definitions {
	return n.Model.Define(n,
		attrs.NewField(n, "ID", &attrs.FieldConfig{
			HelpText: trans.S("The unique identifier for the site."),
			Primary:  true,
			Column:   "id",
			ReadOnly: true,
		}),
		attrs.NewField(n, "Identifier", &attrs.FieldConfig{
			HelpText:  trans.S("The identifier (admin uid) that the LDAP client will use to connect."),
			Column:    "identifier",
			Null:      false,
			Blank:     false,
			MinLength: 4,
			MaxLength: 64,
			FormField: func(opts ...func(fields.Field)) fields.Field {
				base := make([]func(fields.Field), 0)
				base = append(base,
					fields.MinLength(4),
					fields.MaxLength(64),
					fields.Regex(`[a-zA-Z_-]`),
				)
				return fields.CharField(append(base, opts...)...)
			},
		}),
		attrs.NewField(n, "Token", &attrs.FieldConfig{
			HelpText:  trans.S("The token (admin password) that the LDAP client will use to connect."),
			Column:    "token",
			Null:      false,
			Blank:     false,
			MinLength: 32,
			MaxLength: 256,
		}),
		attrs.NewField(n, "CreatedAt", &attrs.FieldConfig{
			HelpText: trans.S("When this service account was created."),
			Column:   "created_at",
			ReadOnly: true,
			Null:     false,
			Blank:    true,
		}),
		attrs.NewField(n, "TokenLastGenerated", &attrs.FieldConfig{
			HelpText: trans.S("When this service account's token was last regenerated."),
			Column:   "last_generated_at",
			ReadOnly: true,
			Null:     false,
			Blank:    true,
		}),
	)
}

func (n *ServiceAccount) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	if reflect.TypeOf(obj) != reflect.TypeOf(n) {
		return nil
	}

	return []migrator.Index{
		{Fields: []string{"Identifier"}, Unique: true},
	}
}
