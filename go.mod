module code.vegaprotocol.io/vega

go 1.22.2

toolchain go1.22.4

require (
	code.vegaprotocol.io/quant v0.2.5
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cosmos/iavl v0.20.0
	github.com/cucumber/godog v0.12.5
	github.com/ethereum/go-ethereum v1.14.5
	github.com/fatih/color v1.16.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/golang/mock v1.6.1-0.20220512030613-73266f9366fc
	github.com/golang/protobuf v1.5.4
	github.com/google/btree v1.1.2
	github.com/hashicorp/golang-lru v1.0.2
	github.com/holiman/uint256 v1.2.4
	github.com/imdario/mergo v0.3.15
	github.com/jessevdk/go-flags v1.4.0
	github.com/jinzhu/copier v0.3.5
	github.com/julienschmidt/httprouter v1.3.0
	github.com/mattn/go-isatty v0.0.20
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.19.1
	github.com/rs/cors v1.11.0
	github.com/satori/go.uuid v1.2.0
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.9.0
	github.com/syndtr/goleveldb v1.0.1-0.20220614013038-64ee5596c38a
	go.elastic.co/apm/module/apmhttp v1.8.0
	go.nanomsg.org/mangos/v3 v3.2.1
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.24.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/99designs/gqlgen v0.17.43
	github.com/BurntSushi/toml v1.4.0
	github.com/PaesslerAG/gval v1.0.0
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/adrg/xdg v0.4.0
	github.com/blang/semver/v4 v4.0.0
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/cometbft/cometbft v1.0.0-rc1
	github.com/cometbft/cometbft-db v0.12.0
	github.com/cometbft/cometbft/api v1.0.0-rc.1
	github.com/cucumber/messages-go/v16 v16.0.1
	github.com/dgraph-io/badger/v2 v2.2007.4
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/didip/tollbooth/v7 v7.0.1
	github.com/emirpasic/gods v1.18.1
	github.com/felixge/fgprof v0.9.3
	github.com/fergusstrange/embedded-postgres v1.17.0
	github.com/fullstorydev/grpcui v1.3.0
	github.com/georgysavva/scany v1.0.0
	github.com/google/go-cmp v0.6.0
	github.com/google/go-github/v50 v50.0.0
	github.com/gorilla/websocket v1.5.3
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/inhies/go-bytesize v0.0.0-20220417184213-4913239db9cf
	github.com/ipfs/boxo v0.13.1
	github.com/ipfs/fs-repo-migrations/fs-repo-12-to-13 v1.0.0
	github.com/ipfs/fs-repo-migrations/fs-repo-13-to-14 v1.0.0
	github.com/ipfs/fs-repo-migrations/fs-repo-14-to-15 v1.0.1
	github.com/ipfs/fs-repo-migrations/tools v0.0.0-20240131173452-0dbdc6f2282c
	github.com/ipfs/go-cid v0.4.1
	github.com/ipfs/go-libipfs v0.7.0
	github.com/ipfs/go-log v1.0.5
	github.com/ipfs/interface-go-ipfs-core v0.11.2
	github.com/ipfs/kubo v0.23.0
	github.com/jackc/pgconn v1.13.0
	github.com/jackc/pgtype v1.12.0
	github.com/jackc/pgx/v4 v4.17.2
	github.com/klauspost/compress v1.17.9
	github.com/libp2p/go-libp2p v0.31.0
	github.com/machinebox/graphql v0.2.2
	github.com/mitchellh/mapstructure v1.5.0
	github.com/muesli/cancelreader v0.2.2
	github.com/muesli/termenv v0.11.0
	github.com/multiformats/go-multiaddr v0.12.4
	github.com/oasisprotocol/curve25519-voi v0.0.0-20220708102147-0a8a51822cae
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58
	github.com/pressly/goose/v3 v3.6.1
	github.com/shopspring/decimal v1.3.1
	github.com/sirupsen/logrus v1.9.3
	github.com/soheilhy/cmux v0.1.4
	github.com/tmc/grpc-websocket-proxy v0.0.0-20220101234140-673ab2c3ae75
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/urfave/cli/v2 v2.27.1
	github.com/vegaprotocol/go-slip10 v0.1.0
	github.com/vektah/gqlparser/v2 v2.5.11
	github.com/wk8/go-ordered-map/v2 v2.0.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.21.0
	golang.org/x/term v0.21.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240617180043-68d350f18fd4
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	bazil.org/fuse v0.0.0-20200117225306-7b5117fecadc // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.4.2 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/Jorropo/jsync v1.0.1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/VictoriaMetrics/fastcache v1.12.2 // indirect
	github.com/alecthomas/units v0.0.0-20231202071711-9a357b53e9c9 // indirect
	github.com/alexbrainman/goissue34681 v0.0.0-20191006012335-3fc7a47baff5 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/bits-and-blooms/bitset v1.10.0 // indirect
	github.com/ceramicnetwork/go-dag-jose v0.1.0 // indirect
	github.com/cheggaaa/pb v1.0.29 // indirect
	github.com/cockroachdb/errors v1.11.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/pebble v1.1.0 // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/consensys/gnark-crypto v0.12.1 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cosmos/crypto v0.0.0-20240309083813-82ed2537802e // indirect
	github.com/cosmos/gogoproto v1.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/crackcomm/go-gitignore v0.0.0-20231225121904-e25f5bc08668 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240223125850-b1e8a79f509c // indirect
	github.com/crate-crypto/go-kzg-4844 v1.0.0 // indirect
	github.com/cskr/pubsub v1.0.2 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/badger/v4 v4.2.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elastic/gosigar v0.14.3 // indirect
	github.com/elgris/jsondiff v0.0.0-20160530203242-765b5c24c302 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/ethereum/go-verkle v0.1.1-0.20240306133620-7d920df305f0 // indirect
	github.com/facebookgo/atomicfile v0.0.0-20151019160806-2de1f203e7d5 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fjl/memsize v0.0.2 // indirect
	github.com/flynn/noise v1.1.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20190607065134-2772fd86a8ff // indirect
	github.com/getsentry/sentry-go v0.18.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-pkgz/expirable-cache v0.1.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/flatbuffers v2.0.8+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/google/pprof v0.0.0-20240618054019-d3b898a103f8 // indirect
	github.com/hannahhoward/go-pubsub v0.0.0-20200423002714-8d62886cc36e // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-bexpr v0.1.10 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/holiman/billy v0.0.0-20240216141850-2abb0c79d3c4 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/ipfs/bbloom v0.0.4 // indirect
	github.com/ipfs/go-bitfield v1.1.0 // indirect
	github.com/ipfs/go-block-format v0.2.0 // indirect
	github.com/ipfs/go-blockservice v0.5.2 // indirect
	github.com/ipfs/go-cidutil v0.1.0 // indirect
	github.com/ipfs/go-datastore v0.6.0 // indirect
	github.com/ipfs/go-ds-badger v0.3.0 // indirect
	github.com/ipfs/go-ds-flatfs v0.5.1 // indirect
	github.com/ipfs/go-ds-leveldb v0.5.0 // indirect
	github.com/ipfs/go-ds-measure v0.2.0 // indirect
	github.com/ipfs/go-fs-lock v0.0.7 // indirect
	github.com/ipfs/go-graphsync v0.15.1 // indirect
	github.com/ipfs/go-ipfs-blockstore v1.3.1 // indirect
	github.com/ipfs/go-ipfs-cmds v0.10.2 // indirect
	github.com/ipfs/go-ipfs-delay v0.0.1 // indirect
	github.com/ipfs/go-ipfs-ds-help v1.1.1 // indirect
	github.com/ipfs/go-ipfs-exchange-interface v0.2.1 // indirect
	github.com/ipfs/go-ipfs-pq v0.0.3 // indirect
	github.com/ipfs/go-ipfs-redirects-file v0.1.1 // indirect
	github.com/ipfs/go-ipfs-util v0.0.3 // indirect
	github.com/ipfs/go-ipld-cbor v0.1.0 // indirect
	github.com/ipfs/go-ipld-format v0.6.0 // indirect
	github.com/ipfs/go-ipld-git v0.1.1 // indirect
	github.com/ipfs/go-ipld-legacy v0.2.1 // indirect
	github.com/ipfs/go-log/v2 v2.5.1 // indirect
	github.com/ipfs/go-merkledag v0.11.0 // indirect
	github.com/ipfs/go-metrics-interface v0.0.1 // indirect
	github.com/ipfs/go-path v0.3.1 // indirect
	github.com/ipfs/go-peertaskqueue v0.8.1 // indirect
	github.com/ipfs/go-unixfsnode v1.9.0 // indirect
	github.com/ipfs/go-verifcid v0.0.3 // indirect
	github.com/ipld/go-car v0.6.2 // indirect
	github.com/ipld/go-car/v2 v2.13.1 // indirect
	github.com/ipld/go-codec-dagpb v1.6.0 // indirect
	github.com/ipld/go-ipld-prime v0.21.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/koron/go-ssdp v0.0.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-doh-resolver v0.4.0 // indirect
	github.com/libp2p/go-flow-metrics v0.1.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.4.1 // indirect
	github.com/libp2p/go-libp2p-gostream v0.6.0 // indirect
	github.com/libp2p/go-libp2p-http v0.5.0 // indirect
	github.com/libp2p/go-libp2p-kad-dht v0.25.2 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.6.3 // indirect
	github.com/libp2p/go-libp2p-pubsub v0.9.3 // indirect
	github.com/libp2p/go-libp2p-pubsub-router v0.6.0 // indirect
	github.com/libp2p/go-libp2p-record v0.2.0 // indirect
	github.com/libp2p/go-libp2p-routing-helpers v0.7.3 // indirect
	github.com/libp2p/go-libp2p-xor v0.1.0 // indirect
	github.com/libp2p/go-mplex v0.7.0 // indirect
	github.com/libp2p/go-msgio v0.3.0 // indirect
	github.com/libp2p/go-nat v0.2.0 // indirect
	github.com/libp2p/go-netroute v0.2.1 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/libp2p/go-yamux/v4 v4.0.1 // indirect
	github.com/libp2p/zeroconf/v2 v2.2.0 // indirect
	github.com/linxGnu/grocksdb v1.8.14 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/matryer/is v1.4.0 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/miekg/dns v1.1.61 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.3.1 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multicodec v0.9.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-multistream v0.4.1 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/onsi/ginkgo/v2 v2.19.0 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/petar/GoLLRB v0.0.0-20210522233825-ae3b015fd3e9 // indirect
	github.com/polydawn/refmt v0.89.0 // indirect
	github.com/prometheus/statsd_exporter v0.22.7 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.3.3 // indirect
	github.com/quic-go/quic-go v0.38.1 // indirect
	github.com/quic-go/webtransport-go v0.5.3 // indirect
	github.com/raulk/go-watchdog v1.3.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/samber/lo v1.39.0 // indirect
	github.com/sosodev/duration v1.2.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/status-im/keycard-go v0.2.0 // indirect
	github.com/supranational/blst v0.3.11 // indirect
	github.com/ucarion/urlpath v0.0.0-20200424170820-7ccc79b76bbb // indirect
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/cbor v0.0.0-20171005072247-63513f603b11 // indirect
	github.com/whyrusleeping/cbor-gen v0.1.2 // indirect
	github.com/whyrusleeping/chunker v0.0.0-20181014151217-fe64bd25879f // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/go-sysinfo v0.0.0-20190219211824-4a357d4b90b1 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.52.0 // indirect
	go.opentelemetry.io/otel v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.27.0 // indirect
	go.opentelemetry.io/otel/metric v1.27.0 // indirect
	go.opentelemetry.io/otel/sdk v1.27.0 // indirect
	go.opentelemetry.io/otel/trace v1.27.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/dig v1.17.1 // indirect
	go.uber.org/fx v1.22.0 // indirect
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240617180043-68d350f18fd4 // indirect
	google.golang.org/grpc/examples v0.0.0-20220922155644-a238cebacde4 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	lukechampine.com/blake3 v1.3.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

require (
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/confio/ics23/go v0.9.0 // indirect
	github.com/cucumber/gherkin-go/v19 v19.0.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1
	github.com/elastic/go-licenser v0.3.1 // indirect
	github.com/elastic/go-sysinfo v1.1.1 // indirect
	github.com/elastic/go-windows v1.0.0 // indirect
	github.com/fullstorydev/grpcurl v1.8.6 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/rpc v1.2.0
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/jhump/protoreflect v1.12.1-0.20220721211354-060cc04fc18b // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/petermattis/goid v0.0.0-20180202154549-b0b1615b78e5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/sasha-s/go-deadlock v0.3.1 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/viper v1.19.0
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.elastic.co/apm v1.12.0 // indirect
	go.elastic.co/fastjson v1.1.0 // indirect
	go.etcd.io/bbolt v1.4.0-alpha.0.0.20240404170359-43604f3112c5 // indirect
	go.uber.org/atomic v1.11.0
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
)

replace (
	github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.23.3
	github.com/fergusstrange/embedded-postgres => github.com/vegaprotocol/embedded-postgres v1.13.1-0.20221123183204-2e7a2feee5bb
	github.com/muesli/cancelreader => github.com/vegaprotocol/cancelreader v0.0.0-20230724130739-6f2217a69449
	github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.3.1-uint256
)
