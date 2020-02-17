package handler

import (
	"encoding/json"
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

// Validate validates values.
// If valid, values is converted to JSON and returned.
// If invalid, values is also converted to JSON and returned, with "error" added.
// This is suitable for server rendering because the page has to retain user input
// with error added.
func Validate(validator *validation.Validator, schemaID string, values url.Values) (map[string]interface{}, error) {
	j := make(map[string]interface{})
	// Do not support recurring parameter
	for name := range values {
		j[name] = values.Get(name)
	}
	err := validator.ValidateGoValue(schemaID, j)
	if err != nil {
		originalErr := err

		b, err := json.Marshal(struct {
			Error *skyerr.APIError `json:"error"`
		}{skyerr.AsAPIError(err)})
		if err != nil {
			return nil, errors.WithSecondaryError(originalErr, err)
		}

		var eJSON map[string]interface{}
		err = json.Unmarshal(b, &eJSON)
		if err != nil {
			return nil, errors.WithSecondaryError(originalErr, err)
		}

		j["error"] = eJSON["error"]

		return j, originalErr
	}
	return j, nil
}
