module code.vegaprotocol.io/data-node

go 1.16

require (
	code.vegaprotocol.io/protos v0.42.1-0.20210923113049-6d9549525594
	code.vegaprotocol.io/quant v0.2.5
	code.vegaprotocol.io/vega v0.42.2-0.20210923133536-fc67a82b0bce
	github.com/99designs/gqlgen v0.13.0
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/mock v1.4.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.4.2
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/mwitkow/go-proto-validators v0.3.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/rs/cors v1.7.0
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/zannen/toml v0.3.2
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.nanomsg.org/mangos/v3 v3.2.1
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.2.1-0.20210705145732-aaa563729a0a
