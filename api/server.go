package api

import (
	"log"
	"net/http"
)

func NewServer() {
	println("Starting HTTP server on port 6666...")
	router := NewRouter()

	log.Fatal(http.ListenAndServe(":6666", router))
}
