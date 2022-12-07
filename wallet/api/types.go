package api

type Connection struct {
	// Hostname is the hostname for which the connection is set.
	Hostname string `json:"hostname"`
	// Wallet is the wallet selected by the client for this connection.
	Wallet string `json:"wallet"`
}
