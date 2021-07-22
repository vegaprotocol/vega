module code.vegaprotocol.io/data-node

go 1.16

require (
	code.vegaprotocol.io/quant v0.2.0
	github.com/99designs/gqlgen v0.13.0
	github.com/btcsuite/btcutil v1.0.2 // indirect
	github.com/cenkalti/backoff/v4 v4.0.0
	github.com/dgraph-io/badger/v2 v2.0.3
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/etcd-io/bbolt v1.3.3 // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-logfmt/logfmt v0.5.0 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/mock v1.2.0
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.2-0.20200707131729-196ae77b8a26 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.5
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/holiman/uint256 v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/julienschmidt/httprouter v1.2.0
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mwitkow/go-proto-validators v0.2.0
	github.com/oasisprotocol/curve25519-voi v0.0.0-20210528083545-b12728c4e0d8
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/rs/cors v1.7.0
	github.com/satori/go.uuid v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d // indirect
	github.com/tendermint/tendermint v0.32.1
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/zannen/toml v0.3.2
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.nanomsg.org/mangos/v3 v3.2.1
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.11.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/grpc v1.27.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.2.1-0.20210705145732-aaa563729a0a
