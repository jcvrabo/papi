package namespace

import (
	"fmt"
	"regexp"
	"strings"
)

type TemplateSegment struct {
	Name            string   `json:"name"`
	Required        bool     `json:"required"`
	ValidationRegex string   `json:"validation_regex,omitempty"`
	AllowedValues   []string `json:"allowed_values,omitempty"`
}

type NamingConfig struct {
	TemplatePattern string
	Segments        []TemplateSegment
}

type ValidationError struct {
	Field   string
	Message string
}

func ValidateNameComponents(config *NamingConfig, components map[string]string) (string, []ValidationError) {
	var errors []ValidationError

	for _, seg := range config.Segments {
		val, exists := components[seg.Name]
		if seg.Required && (!exists || val == "") {
			errors = append(errors, ValidationError{
				Field:   seg.Name,
				Message: fmt.Sprintf("segment '%s' is required", seg.Name),
			})
			continue
		}
		if !exists || val == "" {
			continue
		}
		if seg.ValidationRegex != "" {
			matched, err := regexp.MatchString(seg.ValidationRegex, val)
			if err != nil || !matched {
				errors = append(errors, ValidationError{
					Field:   seg.Name,
					Message: fmt.Sprintf("segment '%s' does not match validation pattern", seg.Name),
				})
			}
		}
		if len(seg.AllowedValues) > 0 {
			found := false
			for _, av := range seg.AllowedValues {
				if av == val {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:   seg.Name,
					Message: fmt.Sprintf("segment '%s' must be one of: %s", seg.Name, strings.Join(seg.AllowedValues, ", ")),
				})
			}
		}
	}

	if len(errors) > 0 {
		return "", errors
	}

	name := config.TemplatePattern
	for _, seg := range config.Segments {
		val := components[seg.Name]
		name = strings.ReplaceAll(name, "{"+seg.Name+"}", val)
	}

	return name, nil
}
