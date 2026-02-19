package sqlerr

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	"github.com/lib/pq"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// ErrCode reports the error code for a given error.
// If the error is nil or is not of type *Error it reports sqlerr.Other.
func ErrCode(err error) Code {
	var pgerr *Error
	if errors.As(err, &pgerr) {
		return pgerr.Code
	}
	return Other
}

// ConvertPgError converts a pq.Error to our custom Error type
func ConvertPgError(src *pq.Error) *Error {
	return &Error{
		Code:           MapCode(string(src.Code)),
		Severity:       MapSeverity(src.Severity),
		DatabaseCode:   string(src.Code),
		Message:        src.Message,
		TableName:      src.Table,
		ColumnName:     src.Column,
		DataTypeName:   src.DataTypeName,
		ConstraintName: src.Constraint,
		driverErr:      src,
	}
}

// generateErrorCode creates consistent error codes from database errors
func generateErrorCode(tableName string, errType Code) string {
	if tableName == "" {
		tableName = "RECORD"
	}

	domain := strings.ToUpper(tableName)
	// Singularize the table name
	if strings.HasSuffix(domain, "S") && len(domain) > 1 {
		domain = domain[:len(domain)-1]
	}

	action := "ERROR"
	switch errType {
	case ForeignKeyViolation:
		action = "NOT_FOUND"
	case UniqueViolation:
		action = "ALREADY_EXISTS"
	case NotNullViolation:
		action = "REQUIRED"
	case CheckViolation:
		action = "INVALID"
	}

	return fmt.Sprintf("%s_%s", domain, action)
}

// formatUserFriendlyMessage generates a user-friendly error message
func formatUserFriendlyMessage(sqlErr *Error) string {
	entityName := getEntityName(sqlErr.TableName, sqlErr.ColumnName)

	switch sqlErr.Code {
	case ForeignKeyViolation:
		return fmt.Sprintf("The referenced %s does not exist", entityName)
	case UniqueViolation:
		return fmt.Sprintf("A %s with this identifier already exists", entityName)
	case NotNullViolation:
		fieldName := humanizeText(sqlErr.ColumnName)
		if fieldName == "" {
			fieldName = "field"
		}
		return fmt.Sprintf("The %s is required", fieldName)
	case CheckViolation:
		fieldName := humanizeText(sqlErr.ColumnName)
		if fieldName != "" {
			return fmt.Sprintf("The %s value does not meet required conditions", fieldName)
		}
		return "One or more values do not meet required conditions"
	default:
		return "An error occurred while processing your request"
	}
}

// getEntityName extracts entity name from database information with consistent rules
func getEntityName(tableName, columnName string) string {
	// First priority: column name logic (most reliable for FK relationships)
	if columnName != "" && strings.HasSuffix(strings.ToLower(columnName), "_id") {
		entity := strings.TrimSuffix(strings.ToLower(columnName), "_id")
		return humanizeText(entity)
	}

	// Second priority: table name (fallback option)
	if tableName != "" {
		// Use singular form
		entity := tableName
		if strings.HasSuffix(entity, "s") && len(entity) > 1 {
			entity = entity[:len(entity)-1]
		}
		return humanizeText(entity)
	}

	// Default fallback
	return "record"
}

// humanizeText converts snake_case to human-readable text
func humanizeText(text string) string {
	if text == "" {
		return ""
	}
	return cases.Title(language.English).String(strings.ReplaceAll(text, "_", " "))
}

// extractColumnForUniqueViolation gets field name from unique constraint
func extractColumnForUniqueViolation(constraintName string) string {
	if constraintName == "" {
		return ""
	}

	// Try standard naming convention first (unique_table_column)
	if strings.HasPrefix(constraintName, "unique_") {
		parts := strings.Split(constraintName, "_")
		if len(parts) >= 3 {
			return parts[len(parts)-1]
		}
	}

	// Try alternate convention (table_column_key)
	re := regexp.MustCompile(`_([^_]+)_(?:key|ukey)$`)
	matches := re.FindStringSubmatch(constraintName)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// HandleError processes a database error into an appropriate application error
func HandleError(err error) error {
	if err == nil {
		return nil
	}
	// If it's already a custom HTTP error, just return it
	var httpErr *errs.ErrorResponse
	if errors.As(err, &httpErr) {
		return err
	}

	// Handle pq specific errors
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		sqlErr := ConvertPgError(pqErr)

		// Generate an appropriate message
		userMessage := formatUserFriendlyMessage(sqlErr)

		switch sqlErr.Code {
		case ForeignKeyViolation:
			return errs.NewBadRequestError(userMessage, false, nil, nil)

		case UniqueViolation:
			columnName := extractColumnForUniqueViolation(sqlErr.ConstraintName)
			if columnName != "" {
				userMessage = strings.ReplaceAll(userMessage, "identifier", humanizeText(columnName))
			}
			return errs.NewBadRequestError(userMessage, true, nil, nil)

		case NotNullViolation:
			fieldErrors := []errs.FieldError{
				{
					Field: strings.ToLower(sqlErr.ColumnName),
					Error: "is required",
				},
			}
			return errs.NewBadRequestError(userMessage, true, fieldErrors, nil)

		case CheckViolation:
			return errs.NewBadRequestError(userMessage, true, nil, nil)

		default:
			return errs.NewInternalServerError()
		}
	}

	// Handle common not-found errors
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, sql.ErrNoRows):
		errMsg := err.Error()
		tablePrefix := "table:"
		if strings.Contains(errMsg, tablePrefix) {
			table := strings.Split(strings.Split(errMsg, tablePrefix)[1], ":")[0]
			entityName := getEntityName(table, "")
			return errs.NewNotFoundError(fmt.Sprintf("%s not found",
				entityName), true)
		}
		return errs.NewNotFoundError("Resource not found", false)
	}

	return errs.NewInternalServerError()
}
