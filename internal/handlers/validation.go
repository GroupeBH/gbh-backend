package handlers

import (
	"github.com/go-playground/validator/v10"
)

func validationDetails(errs validator.ValidationErrors) map[string]string {
	details := make(map[string]string, len(errs))
	for _, err := range errs {
		field := err.Field()
		details[field] = err.Tag()
	}
	return details
}
