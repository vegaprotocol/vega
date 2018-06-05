// +build integration

package rpc

import (
	"os"
	"testing"

	"github.com/tendermint/tendermint/blob/master/rpc/lib/server"
	"github.com/tendermint/tmlibs/log"
)

var client *Client

func initClient() {
	client = &Client{}
}

func initServer() {
	s := server.NewRPCFunc()
	// NewWSRPCFunc
	// RegisterRPCFuncs
}

func TestHealth(t *testing.T) {
}

func TestMain(m *testing.M) {
	_ = log.NewNopLogger()
	client = &Client{}
	client.Connect()
	retval := m.Run()
	os.Exit(retval)
}
