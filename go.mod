module code.vegaprotocol.io/data-node

go 1.16

require (
	code.vegaprotocol.io/protos v0.0.0-20210729134731-59b61c04a76e
	code.vegaprotocol.io/quant v0.2.0
	github.com/99designs/gqlgen v0.13.0
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.2-0.20200707131729-196ae77b8a26 // indirect
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/holiman/uint256 v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/mwitkow/go-proto-validators v0.2.0
	github.com/oasisprotocol/curve25519-voi v0.0.0-20210528083545-b12728c4e0d8
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/rs/cors v1.7.0
	github.com/satori/go.uuid v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/zannen/toml v0.3.2
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.nanomsg.org/mangos/v3 v3.2.1
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.2.1-0.20210705145732-aaa563729a0a
