package rest

import (
	"fmt"
	"html"
	"net/http"
)

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there, %q", html.EscapeString(r.URL.Path))
}
