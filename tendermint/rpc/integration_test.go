// +build integration

package rpc

import (
	"os"
	"testing"
	"time"
)

var client *Client

func TestHealth(t *testing.T) {
}

func TestMain(m *testing.M) {
	client = &Client{
		Address:          DefaultAddress,
		HandshakeTimeout: 5 * time.Second,
		WriteTimeout:     5 * time.Second,
	}
	client.Connect()
	retval := m.Run()
	os.Exit(retval)
}
