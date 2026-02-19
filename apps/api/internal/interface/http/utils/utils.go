package utils

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
)

func ParseUUIDParam(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errs.NewBadRequestError("invalid id provided", true, []errs.FieldError{{Field: "id", Error: "must be a valid uuid"}}, nil)
	}
	return id, nil
}

func ParseQueryInt(raw string, maxAndDefaultVal ...int) int {
	var (
		defaultVal int
		max        *int
	)

	if len(maxAndDefaultVal) > 0 {
		max = &maxAndDefaultVal[0]
	}
	if len(maxAndDefaultVal) > 1 {
		defaultVal = maxAndDefaultVal[1]
	}

	if raw == "" {
		return defaultVal
	}

	if v, err := strconv.Atoi(raw); err == nil {
		if v < 1 {
			return defaultVal
		}
		if max != nil && v > *max {
			return *max
		}
		return v
	}

	return defaultVal
}
