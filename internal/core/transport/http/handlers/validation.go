package handlers

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

func init() {
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})
}

func FormatValidation(valErrs validator.ValidationErrors) map[string]string {
	messages := make(map[string]string, len(valErrs))

	for _, fe := range valErrs {
		field := fe.Field()

		switch fe.Tag() {
		case "required":
			messages[field] = "Field is required"
		case "min":
			messages[field] = fmt.Sprintf("Field must be at least %s", fe.Param())
		case "max":
			messages[field] = fmt.Sprintf("Field must be at most %s", fe.Param())
		case "url":
			messages[field] = "Field must be a valid URL"
		default:
			messages[field] = "Field is invalid"
		}
	}

	return messages
}
