package provider

import (
	"net/url"
)

type ValidateProvider interface {
	// Validate validates form against schemaID.
	// In either case, the form is converted to JSON and
	// be returned as the first result.
	Validate(schemaID string, form url.Values) (map[string]interface{}, error)
}
