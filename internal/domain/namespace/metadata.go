package namespace

import (
	"fmt"
	"regexp"
	"strings"
)

type MetadataRule struct {
	FieldName       string
	IsMandatory     bool
	ValidationRegex string
	AllowedValues   []string
}

func ValidateMetadata(rules []MetadataRule, metadata map[string]string) []ValidationError {
	var errors []ValidationError

	for _, rule := range rules {
		if !rule.IsMandatory {
			continue
		}
		val, exists := metadata[rule.FieldName]
		if !exists || val == "" {
			errors = append(errors, ValidationError{
				Field:   rule.FieldName,
				Message: fmt.Sprintf("metadata field '%s' is mandatory", rule.FieldName),
			})
			continue
		}
		if rule.ValidationRegex != "" {
			matched, err := regexp.MatchString(rule.ValidationRegex, val)
			if err != nil || !matched {
				errors = append(errors, ValidationError{
					Field:   rule.FieldName,
					Message: fmt.Sprintf("metadata field '%s' does not match validation pattern", rule.FieldName),
				})
			}
		}
		if len(rule.AllowedValues) > 0 {
			found := false
			for _, av := range rule.AllowedValues {
				if av == val {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:   rule.FieldName,
					Message: fmt.Sprintf("metadata field '%s' must be one of: %s", rule.FieldName, strings.Join(rule.AllowedValues, ", ")),
				})
			}
		}
	}

	return errors
}
