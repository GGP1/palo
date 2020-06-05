package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Respond is the protocol function for plain text resposes
func Respond(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "plain/text")

	w.WriteHeader(status)

	if _, err := io.Copy(w, &buf); err != nil {
		fmt.Println("Respond: ", err)
	}
}

// RespondJSON is the protocol function for JSON responses
func RespondJSON(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(status)

	if _, err := io.Copy(w, &buf); err != nil {
		fmt.Println("Respond: ", err)
	}
}
