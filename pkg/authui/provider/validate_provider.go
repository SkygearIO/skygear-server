package provider

import (
	"net/url"
)

type ValidateProvider interface {
	// Validate validates form against schemaID.
	// In either case, the form is converted to JSON and
	// be returned as the first result.
	// If form is invalid, the returned JSON has "error" added.
	// This is suitable for server rendering because the page has to retain user input
	// with error added.
	Validate(schemaID string, form url.Values) (map[string]interface{}, error)
}
