package store

import (
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	uuid "github.com/satori/go.uuid"
)

type Config struct {
	// Mandatory Setting, must be set
	IDSeed string `long:"id-seed" description:"used to generate the id and Key pair for this node"`

	// Optional Settings
	UseIpfsDefaultPeers encoding.Bool `long:"use-ipfs-default-peers" description:"if true ipfs default peers will be appended to the bootstrap peers"`
	BootstrapPeers      []string      `long:"bootstrap-peers" description:"a list of the multiaddress of bootstrap peers, will be used in addition to the ipfs default peers if enabled"`
	SwarmPort           int           `long:"swarm-port" description:"ipfs swarm port"`

	// Without this there would be no way to isolate an environment if needed and process a given chains data (e.g. for dev)
	SwarmKeyOverride string `long:"swarm-key-override" description:"optional swarm key override, the default behaviour is to use the datanode's chain id'"`

	StartWebUI encoding.Bool `long:"start-web-ui" description:"if true the store will expose the ipfs web UI"`
	WebUIPort  int           `long:"webui-port" description:"webui port"`
}

func NewDefaultConfig() Config {
	return Config{
		IDSeed:              uuid.NewV4().String(),
		BootstrapPeers:      []string{},
		UseIpfsDefaultPeers: true,

		SwarmPort: 4001,

		StartWebUI: false,
		WebUIPort:  5001,
	}
}
