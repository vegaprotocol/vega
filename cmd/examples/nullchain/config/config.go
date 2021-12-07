package config

import "time"

const (
	TimeforwardAddress = "http://localhost:3101/api/v1/forwardtime"
	FaucetAddress      = "http://localhost:1790/api/v1/mint"
	GRCPAddress        = "localhost:3007"
	GoveranceAsset     = "VOTE"
	NormalAsset        = "XYZ"
	WalletFolder       = "YOU_PATH_TO/vegahome/data/wallets"
	Passphrase         = "pin"
	BlockDuration      = time.Second
)
