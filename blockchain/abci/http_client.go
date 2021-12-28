package abci

import (
	"net/http"
	"time"
)

func defaultHTTPClient(remoteAddr string) (*http.Client, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			// Set to true to prevent GZIP-bomb DoS attacks
			DisableCompression: true,
			DisableKeepAlives:  true,
			MaxIdleConns:       100,
			IdleConnTimeout:    5 * time.Second,
		},
	}

	return client, nil
}
