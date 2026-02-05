package validation

import (
	"reflect"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	v *validator.Validate
}

func New() *Validator {
	v := validator.New()

	v.RegisterValidation("date", func(fl validator.FieldLevel) bool {
		value, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		_, err := time.Parse("2006-01-02", value)
		return err == nil
	})

	v.RegisterValidation("clock", func(fl validator.FieldLevel) bool {
		value, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		_, err := time.Parse("15:04", value)
		return err == nil
	})

	phoneRegex := regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		value, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return phoneRegex.MatchString(value)
	})

	v.RegisterValidation("minutes15", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind() != reflect.Int && fl.Field().Kind() != reflect.Int32 && fl.Field().Kind() != reflect.Int64 {
			return false
		}
		val := fl.Field().Int()
		return val > 0 && val%15 == 0
	})

	return &Validator{v: v}
}

func (v *Validator) Struct(s interface{}) error {
	return v.v.Struct(s)
}

func (v *Validator) ValidationErrors(err error) validator.ValidationErrors {
	if err == nil {
		return nil
	}
	if ve, ok := err.(validator.ValidationErrors); ok {
		return ve
	}
	return nil
}
