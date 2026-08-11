package validation

import (
	"CrudTutorialProject/internal/apperror"
	"errors"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tagName := range []string{"json", "query", "yaml"} {
			name := strings.SplitN(field.Tag.Get(tagName), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}

		return field.Name
	})

	return &Validator{
		validate: v,
	}
}

func (v *Validator) Struct(value any) error {
	if err := v.validate.Struct(value); err != nil {
		return toFieldValidationError(err)
	}

	return nil
}

func toFieldValidationError(err error) error {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}

	fields := make(map[string]string, len(validationErrors))

	for _, fieldErr := range validationErrors {
		fieldName := fieldErr.Field()
		fields[fieldName] = messageForTag(fieldErr)
	}

	return apperror.NewFieldValidationError(fields)
}

func messageForTag(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + err.Param() + " characters"
	case "max":
		return "must be at most " + err.Param() + " characters"
	case "email":
		return "must be a valid email"
	case "gte":
		return "must be greater than or equal to " + err.Param()
	case "lte":
		return "must be less than or equal to " + err.Param()
	case "oneof":
		return "must be one of: " + err.Param()
	default:
		return "is invalid"
	}
}
