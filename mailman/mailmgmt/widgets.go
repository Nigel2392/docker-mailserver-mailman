package mailmgmt

import (
	"context"
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
	bytesize "github.com/inhies/go-bytesize"
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

type BytesizeInput[output constraints.Unsigned] struct {
	*widgets.BaseWidget
}

var parseRegex = regexp.MustCompile(`^(0+|(\d+(|\.\d+))\s*[a-zA-Z]{1,3})$`)

func NewByteSizeInput[T constraints.Unsigned](attrs map[string]string) widgets.Widget {
	return &BytesizeInput[T]{widgets.NewBaseWidget("text", "forms/widgets/input.html", attrs)}
}

func (n *BytesizeInput[T]) ValueToGo(value interface{}) (out interface{}, err error) {
	if value == nil {
		return 0, nil
	}

	var this = reflect.TypeFor[T]()
	var rv = reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = rv.Convert(this).Interface()
	case reflect.String:
		v := rv.String()

		if !parseRegex.MatchString(v) {
			return v, errors.Wrap(
				errs.ErrInvalidSyntax,
				"Invalid syntax for field, examples are: 0, 100kb, 10mb, 1GB",
			)
		}

		i, err := strconv.Atoi(v)

		switch {
		case err != nil && !errors.Is(err, strconv.ErrSyntax):
			return v, errors.Wrap(
				errs.ErrInvalidSyntax,
				"Invalid syntax for field, examples are: 0, 100kb, 10mb, 1GB",
			)

		case err == nil && i > 0:
			return v, errors.Wrap(
				errs.ErrInvalidSyntax,
				"Invalid syntax for field, examples are: 0, 100kb, 10mb, 1GB",
			)

		case err == nil && i == 0:
			value = T(0)
		default:
			value, err = bytesize.Parse(rv.String())
			value = reflect.ValueOf(value).Convert(this).Interface()
		}

	default:
		err = errors.Wrapf(
			errs.ErrInvalidType,
			"unknown type %T", value,
		)
	}

	return value, err
}

var _uint64_t = reflect.TypeOf(uint64(0))

func (n *BytesizeInput[T]) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return value
	}

	switch v := value.(type) {
	case string:
		return v
	case T:
		return bytesize.ByteSize(v).String()
	default:
		rv := reflect.ValueOf(v)
		rv = rv.Convert(_uint64_t)
		return bytesize.ByteSize(rv.Uint()).String()
	}
}

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
