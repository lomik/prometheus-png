package pkg

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/form"
)

var formDecoder *form.Decoder

func init() {
	formDecoder = form.NewDecoder()

	var duration time.Duration

	formDecoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		return time.ParseDuration(vals[0])
	}, duration)
}

func parseRequest(w http.ResponseWriter, r *http.Request, params interface{}, required ...string) bool {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}

	for i := 0; i < len(required); i++ {
		if r.FormValue(required[i]) == "" {
			http.Error(w, fmt.Sprintf("%s is required", required[i]), http.StatusBadRequest)
			return false
		}
	}

	if err := formDecoder.Decode(&params, r.Form); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}

	return true
}

func parsePostRequest(w http.ResponseWriter, r *http.Request, params interface{}, required ...string) bool {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusBadRequest)
		return false
	}

	return parseRequest(w, r, params, required...)
}

func parseGetRequest(w http.ResponseWriter, r *http.Request, params interface{}, required ...string) bool {
	if r.Method != "GET" {
		http.Error(w, "GET required", http.StatusBadRequest)
		return false
	}

	return parseRequest(w, r, params, required...)
}
