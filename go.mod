module code.vegaprotocol.io/vega

go 1.16

require (
	code.vegaprotocol.io/oracles-relay v0.0.0-20210201140234-f047e1bf6df3
	code.vegaprotocol.io/quant v0.2.5
	github.com/99designs/gqlgen v0.11.3
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v4 v4.0.0
	github.com/cucumber/godog v0.11.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/ethereum/go-ethereum v1.9.25
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/google/btree v1.0.0
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/holiman/uint256 v1.2.0
	github.com/imdario/mergo v0.3.11
	github.com/jessevdk/go-flags v1.4.0
	github.com/jinzhu/copier v0.2.8
	github.com/julienschmidt/httprouter v1.3.0
	github.com/mwitkow/go-proto-validators v0.2.0
	github.com/oasisprotocol/curve25519-voi v0.0.0-20210716083614-f38f8e8b0b84
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/rs/cors v1.7.0
	github.com/satori/go.uuid v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.1.2
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.11
	github.com/vektah/gqlparser/v2 v2.0.1
	github.com/zannen/toml v0.3.2
	go.elastic.co/apm v1.12.0 // indirect
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0-rc.1
)

replace github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.2.1-0.20210705145732-aaa563729a0a
