package parsers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/azizndao/glib/errs"
	"github.com/azizndao/glib/utils/fsutil"
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

func ParseQuery(request *http.Request) (map[string]any, error) {
	queryParams, err := url.ParseQuery(request.URL.RawQuery)
	if err == nil {
		query := make(map[string]any, len(queryParams))
		flatten(query, queryParams)
		return query, nil
	}
	return nil, err
}

func GenerateFlatMap(request *http.Request, maxSize int64) (map[string]any, error) {
	flatMap := make(map[string]any)
	request.Form = url.Values{} // Prevent Form from being parsed because it would be redundant with our parsing
	err := request.ParseMultipartForm(maxSize)
	if err != nil {
		if err == http.ErrNotMultipart {
			if err := request.ParseForm(); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if request.PostForm != nil {
		flatten(flatMap, request.PostForm)
	}
	if request.MultipartForm != nil {
		flatten(flatMap, request.MultipartForm.Value)

		for field, headers := range request.MultipartForm.File {
			files, err := fsutil.ParseMultipartFiles(headers)
			if err != nil {
				return nil, err
			}
			flatMap[field] = files
		}
	}

	// Source form is not needed anymore, clear it.
	request.Form = nil
	request.PostForm = nil
	request.MultipartForm = nil

	return flatMap, nil
}

func flatten(dst map[string]any, values url.Values) {
	for field, value := range values {
		if len(value) > 1 {
			dst[field] = value
		} else {
			dst[field] = value[0]
		}
	}
}
