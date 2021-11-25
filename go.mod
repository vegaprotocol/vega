module code.vegaprotocol.io/vega

go 1.16

require (
	code.vegaprotocol.io/oracles-relay v0.0.0-20210201140234-f047e1bf6df3
	code.vegaprotocol.io/protos v0.46.1-0.20211125092808-8c9433d9736d
	code.vegaprotocol.io/quant v0.2.5
	code.vegaprotocol.io/shared v0.0.0-20211015074835-9ed837d93090
	code.vegaprotocol.io/vegawallet v0.9.2
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cosmos/iavl v0.15.3
	github.com/cucumber/godog v0.11.0
	github.com/cucumber/messages-go/v10 v10.0.3
	github.com/ethereum/go-ethereum v1.9.25
	github.com/fatih/color v1.7.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/mock v1.4.3
	github.com/golang/protobuf v1.5.2
	github.com/google/btree v1.0.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/holiman/uint256 v1.2.0
	github.com/imdario/mergo v0.3.11
	github.com/jessevdk/go-flags v1.4.0
	github.com/jinzhu/copier v0.2.8
	github.com/julienschmidt/httprouter v1.3.0
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.14
	github.com/mwitkow/go-proto-validators v0.3.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/rs/cors v1.7.0
	github.com/satori/go.uuid v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/spf13/afero v1.1.2
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tendermint/tendermint v0.34.14
	github.com/tendermint/tm-db v0.6.4
	go.elastic.co/apm v1.12.0 // indirect
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.nanomsg.org/mangos/v3 v3.2.1
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	google.golang.org/genproto v0.0.0-20210611144927-798beca9d670 // indirect
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.2.1-0.20210705145732-aaa563729a0a
