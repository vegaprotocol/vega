module code.vegaprotocol.io/vega

require (
	code.vegaprotocol.io/quant v0.1.0
	github.com/99designs/gqlgen v0.10.1
	github.com/DATA-DOG/godog v0.7.13
	github.com/blang/semver v3.5.1+incompatible
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32 // indirect
	github.com/dgraph-io/badger v1.6.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.4.7
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.3.2
	github.com/google/btree v1.0.0
	github.com/google/protobuf v3.7.0+incompatible // indirect
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/websocket v1.4.1
	github.com/grpc-ecosystem/grpc-gateway v1.9.0
	github.com/mwitkow/go-proto-validators v0.2.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/rs/cors v1.7.0
	github.com/satori/go.uuid v1.2.0
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/stumble/gorocksdb v0.0.3 // indirect
	github.com/tendermint/tendermint v0.32.6
	github.com/vegaprotocol/modvendor v0.0.2 // indirect
	github.com/vektah/gqlparser v1.1.2
	github.com/zannen/toml v0.3.2
	go.elastic.co/apm/module/apmhttp v1.5.0
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2
	google.golang.org/grpc v1.23.1
)

replace (
	code.vegaprotocol.io/quant => gitlab.com/vega-protocol/quant v0.1.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.2
)
