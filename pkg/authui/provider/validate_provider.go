package provider

import (
	"net/url"
)

type ValidateProvider interface {
	// Prevalidate remove empty form values and set default values.
	// When a form is submitted and a text field is empty,
	// the form will have that field with empty string value,
	// making the JSON schema keyword required useless.
	Prevalidate(form url.Values)
	// Validate validates form against schemaID.
	// In either case, the form is converted to JSON and
	// be returned as the first result.
	Validate(schemaID string, form url.Values) (map[string]interface{}, error)
}
