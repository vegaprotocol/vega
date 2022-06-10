module code.vegaprotocol.io/data-node

go 1.18

require (
	code.vegaprotocol.io/protos v0.51.2-0.20220609143205-3f2e3b7dcdc7
	code.vegaprotocol.io/quant v0.2.5
	code.vegaprotocol.io/shared v0.0.0-20220321185018-3b5684b00533
	code.vegaprotocol.io/vega v0.51.2-0.20220607090205-1cbbb9aba7d0
	github.com/99designs/gqlgen v0.16.0
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/dgraph-io/badger/v2 v2.2007.3
	github.com/fergusstrange/embedded-postgres v0.0.0-00010101000000-000000000000
	github.com/fsnotify/fsnotify v1.5.1
	github.com/fullstorydev/grpcui v1.2.0
	github.com/georgysavva/scany v0.3.0
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7
	github.com/gorilla/websocket v1.5.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.9.0
	github.com/jackc/pgconn v1.12.1
	github.com/jackc/pgtype v1.11.0
	github.com/jackc/pgx/v4 v4.14.1
	github.com/jessevdk/go-flags v1.4.0
	github.com/machinebox/graphql v0.2.2
	github.com/mwitkow/go-proto-validators v0.3.2
	github.com/pkg/errors v0.9.1
	github.com/pressly/goose/v3 v3.5.1
	github.com/prometheus/client_golang v1.12.1
	github.com/rs/cors v1.8.2
	github.com/satori/go.uuid v1.2.0
	github.com/shopspring/decimal v1.2.0
	github.com/stretchr/testify v1.7.1
	github.com/vektah/gqlparser/v2 v2.2.0
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.nanomsg.org/mangos/v3 v3.2.1
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.27.1
)

require golang.org/x/exp v0.0.0-20220428152302-39d4317da171 // indirect

require (
	code.vegaprotocol.io/vegawallet v0.15.2-0.20220529200156-cb58876f94ed // indirect
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/DataDog/zstd v1.4.1 // indirect
	github.com/adrg/xdg v0.4.0 // indirect
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/confio/ics23/go v0.6.3 // indirect
	github.com/cosmos/iavl v0.15.3 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v0.0.3-0.20200630154024-f66de99634de // indirect
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/elastic/go-licenser v0.3.1 // indirect
	github.com/elastic/go-sysinfo v1.1.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.10.16 // indirect
	github.com/fullstorydev/grpcurl v1.8.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/holiman/uint256 v1.2.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/puddle v1.2.1 // indirect
	github.com/jhump/protoreflect v1.10.1 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/lib/pq v1.10.4 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/oasisprotocol/curve25519-voi v0.0.0-20220317090546-adb2f9614b17 // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/sasha-s/go-deadlock v0.2.1-0.20190427202633-1595213edefa // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tecbot/gorocksdb v0.0.0-20191217155057-f0fad39f321c // indirect
	github.com/tendermint/tendermint v0.34.15 // indirect
	github.com/tendermint/tm-db v0.6.6 // indirect
	github.com/urfave/cli/v2 v2.3.0 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.elastic.co/apm v1.12.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.6.0-dev.0.20211013180041-c96bc1413d57 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sys v0.0.0-20220318055525-2edf467146b5 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.9 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gonum.org/v1/gonum v0.9.1 // indirect
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

replace github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.2.1-0.20210705145732-aaa563729a0a

replace github.com/fergusstrange/embedded-postgres => github.com/vegaprotocol/embedded-postgres v1.13.1-0.20220607151211-5f2f488de508

replace github.com/jackc/pgx/v4 v4.14.1 => github.com/pscott31/pgx/v4 v4.16.2-0.20220531164027-bd666b84b61f
