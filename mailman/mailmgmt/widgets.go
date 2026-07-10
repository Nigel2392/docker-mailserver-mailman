package mailmgmt

import (
	"context"
	"fmt"
	"net/mail"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
)

type UsernameWidget struct {
	*widgets.MultiWidget
}

func getCleanedEmail(cleanedData map[string]interface{}, fieldName string) (*mail.Address, error) {
	var emailDomain, ok = cleanedData[fieldName].(map[string]interface{})
	if !ok {
		return nil, errs.ErrFieldRequired
	}

	domainValue, ok := emailDomain["domain"]
	if !ok {
		return nil, errs.ErrFieldRequired
	}
	usernameValue, ok := emailDomain["username"]
	if !ok {
		return nil, errs.ErrFieldRequired
	}

	domainRow, err := queries.
		GetQuerySet(&Domain{}).
		Filter("ID", domainValue).
		Get()
	if err != nil {
		return nil, err
	}

	return mail.ParseAddress(fmt.Sprintf(
		"%s@%s", usernameValue, domainRow.Object.Domain,
	))
}

func NewEmailDomainWidget(attrs map[string]string) *UsernameWidget {
	var w = widgets.NewMultiWidget(attrs)
	w.AddWidget("username", widgets.NewTextInput(nil))
	w.AddWidget("domain", formfields.ModelSelectWidget(
		false, "--------",
		chooser.BaseChooserOptions{
			TargetObject: &Domain{},
			GetPrimaryKey: func(_ context.Context, i interface{}) interface{} {
				var def, ok = i.(*Domain)
				if !ok {
					assert.Fail("object %T is not a Definer", i)
				}
				return def.ID
			},
		},
		nil,
	))
	return &UsernameWidget{
		MultiWidget: w,
	}
}
