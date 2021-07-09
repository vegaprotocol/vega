package http

import (
	"fmt"
	"net"
	"net/http"
)

// RemoteAddr collects possible remote addresses from a request
func RemoteAddr(r *http.Request) (string, error) {
	// Only defined when site is accessed via non-anonymous proxy
	// and takes precedence over RemoteAddr
	remote := r.Header.Get("X-Forwarded-For")
	if remote != "" {
		return remote, nil
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("unable to get remote address (failed to split host:port) from \"%s\": %v", r.RemoteAddr, err)
	}

	return ip, nil
}

