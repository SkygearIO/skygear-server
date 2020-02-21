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
	Validate(schemaID string, formJSON map[string]interface{}) error
}

func FormToJSON(form url.Values) map[string]interface{} {
	j := make(map[string]interface{})
	// Do not support recurring parameter
	for name := range form {
		j[name] = form.Get(name)
	}
	return j
}
