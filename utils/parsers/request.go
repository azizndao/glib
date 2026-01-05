package parsers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/azizndao/glib/errs"
)

// GetQuerySlice extracts all values for a query parameter.
// Returns an empty slice if the parameter doesn't exist.
func GetQuerySlice(r *http.Request, name string) []string {
	values := r.URL.Query()[name]
	if values == nil {
		return []string{}
	}
	return values
}

// ParseJSONBody decodes the JSON request body into type T.
// Returns a structured error if decoding fails.
func ParseJSONBody[T any](r *http.Request) (T, error) {
	var result T
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		return result, errs.B().
			Code(errs.InvalidArgument).
			Msg(fmt.Sprintf("invalid JSON request body: %v", err)).
			Cause(err).
			Err()
	}
	return result, nil
}
