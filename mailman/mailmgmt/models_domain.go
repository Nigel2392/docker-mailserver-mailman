package mailmgmt

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

//	return queries.GetQuerySet(&Domain{}).
//		WithContext(ctx).
//		Filter("Domain__iexact", domain)

type Domain struct {
	models.Model `table:"domains"`
	ID           int64
	Name         string
	Domain       string
}

func (n *Domain) FieldDefs() attrs.Definitions {
	return n.Model.Define(n, n.Fields)
}

func (n *Domain) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	if reflect.TypeOf(obj) != reflect.TypeOf(n) {
		return nil
	}

	return []migrator.Index{
		{Fields: []string{"Domain"}, Unique: true},
	}
}

func (n *Domain) Fields(d attrs.Definer) []attrs.Field {
	return []attrs.Field{
		attrs.NewField(n, "ID", &attrs.FieldConfig{
			HelpText: trans.S("The unique identifier for the site."),
			Primary:  true,
			Column:   "id",
			ReadOnly: true,
		}),
		attrs.NewField(n, "Name", &attrs.FieldConfig{
			HelpText:  trans.S("The name of the site."),
			Column:    "site_name",
			Null:      false,
			Blank:     false,
			MinLength: 2,
			MaxLength: 64,
		}),
		attrs.NewField(n, "Domain", &attrs.FieldConfig{
			HelpText:  trans.S("The domain of the site, e.g. example.com."),
			Column:    "domain",
			Null:      false,
			Blank:     false,
			MinLength: 2,
			MaxLength: 256,
		}),
	}
}

func (n *Domain) BeforeSave(ctx context.Context) error {
	if n.Name == "" {
		n.Name = "Default Site"
	}

	if n.Domain == "" {
		n.Domain = "localhost"
	}

	return nil
}
