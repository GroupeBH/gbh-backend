package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

func DecodeJSON(body io.Reader, v interface{}) error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must contain a single JSON object")
	}
	return nil
}

func ValidationDetails(errs validator.ValidationErrors) map[string]string {
	if len(errs) == 0 {
		return nil
	}
	details := make(map[string]string, len(errs))
	for _, err := range errs {
		details[err.Field()] = err.Tag()
	}
	return details
}

func ParseLimitOffset(values url.Values, defaultLimit, maxLimit int64) (int64, int64, error) {
	limit := defaultLimit
	offset := int64(0)

	rawLimit := strings.TrimSpace(values.Get("limit"))
	if rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, 0, errors.New("invalid limit")
		}
		limit = parsed
	}

	rawOffset := strings.TrimSpace(values.Get("offset"))
	if rawOffset != "" {
		parsed, err := strconv.ParseInt(rawOffset, 10, 64)
		if err != nil || parsed < 0 {
			return 0, 0, errors.New("invalid offset")
		}
		offset = parsed
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	return limit, offset, nil
}
