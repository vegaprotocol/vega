package api

import (
	"log"
	"net/http"
)

func NewServer() {
	println("Starting HTTP server on port 8080...")
	router := NewRouter()

	log.Fatal(http.ListenAndServe(":8080", router))
}
