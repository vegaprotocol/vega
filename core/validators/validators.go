package validators

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/validators NodeWallets,TimeService,Commander,ValidatorTopology,Wallet,ValidatorPerformance,Notary,Signatures,MultiSigTopology
