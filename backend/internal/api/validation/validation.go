// Package validation checks the requests of the API with the `validate` tags of
// the models, and tells the client which field is wrong.
package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/diyorbeknematov/lms/pkg/helpers"
	"github.com/diyorbeknematov/lms/pkg/password"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var once sync.Once

// Setup points Gin's validator at the `validate` tag of the models, names the
// fields by their json (or form) name, and adds the rules of this project:
// `password` and `username`. It is safe to call it many times.
func Setup() {
	once.Do(func() {
		engine, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			panic("validation: Gin does not use go-playground/validator")
		}

		engine.SetTagName("validate")

		engine.RegisterTagNameFunc(func(field reflect.StructField) string {
			for _, tag := range []string{"json", "form"} {
				name := strings.Split(field.Tag.Get(tag), ",")[0]
				if name != "" && name != "-" {
					return name
				}
			}

			return field.Name
		})

		_ = engine.RegisterValidation("password", func(level validator.FieldLevel) bool {
			return password.Validate(level.Field().String()) == nil
		})

		_ = engine.RegisterValidation("username", func(level validator.FieldLevel) bool {
			_, err := helpers.NormalizeUsername(level.Field().String())

			return err == nil
		})
	})
}

// Fields turns the error of a bind into the wrong fields and their messages.
// It returns nil when the error is not about the fields (for example a broken
// JSON).
func Fields(err error) map[string]string {
	var invalid validator.ValidationErrors

	if !errors.As(err, &invalid) {
		return nil
	}

	fields := make(map[string]string, len(invalid))

	for _, fieldError := range invalid {
		fields[path(fieldError)] = message(fieldError)
	}

	return fields
}

// path is the name of the field, without the name of the struct in front.
func path(fieldError validator.FieldError) string {
	name := fieldError.Namespace()

	if _, rest, found := strings.Cut(name, "."); found {
		return rest
	}

	return name
}

func message(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "password":
		return "must have at least 8 characters, a capital letter, a small letter, a digit and a special character"
	case "username":
		return "must have 3 to 30 Latin letters, digits, dots, dashes or underscores"
	case "min":
		if fieldError.Kind() == reflect.String {
			return fmt.Sprintf("must have at least %s characters", fieldError.Param())
		}

		return fmt.Sprintf("must be at least %s", fieldError.Param())
	case "max":
		if fieldError.Kind() == reflect.String {
			return fmt.Sprintf("must have at most %s characters", fieldError.Param())
		}

		return fmt.Sprintf("must be at most %s", fieldError.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fieldError.Param())
	case "gte":
		return fmt.Sprintf("must be at least %s", fieldError.Param())
	case "lte":
		return fmt.Sprintf("must be at most %s", fieldError.Param())
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(fieldError.Param(), " ", ", ")
	case "uuid", "uuid4":
		return "must be a valid id"
	}

	return "is invalid"
}
