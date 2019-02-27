module vega

require (
	github.com/99designs/gqlgen v0.7.2
	github.com/AndreasBriese/bbloom v0.0.0-20180913140656-343706a395b7 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32 // indirect
	github.com/dgraph-io/badger v1.5.4
	github.com/dgryski/go-farm v0.0.0-20190104051053-3adb47b1fb0f // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/ethereum/go-ethereum v1.8.22 // indirect
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-kit/kit v0.8.0 // indirect
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.2.0
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/grpc-ecosystem/grpc-gateway v1.7.0
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jmhodges/levigo v0.0.0-20161115193449-c42d9e0ca023 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/rs/cors v1.6.0
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.3.0
	github.com/syndtr/goleveldb v0.0.0-20190203031304-2f17a3356c66 // indirect
	github.com/tendermint/go-amino v0.14.1 // indirect
	github.com/tendermint/tendermint v0.30.0
	github.com/vektah/gqlparser v1.1.1
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd
	golang.org/x/sys v0.0.0-20190222072716-a9d3bda3a223 // indirect
	google.golang.org/grpc v1.18.0
)

replace (
	github.com/dgraph-io/badger => github.com/dgraph-io/badger v1.5.5-0.20190214192501-3196cc1d7a5f
	github.com/golang/protobuf => github.com/golang/protobuf v1.2.1-0.20190205222052-c823c79ea157
)
