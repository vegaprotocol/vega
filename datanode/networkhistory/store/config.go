package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"

	"code.vegaprotocol.io/vega/logging"

	"github.com/ipfs/kubo/config"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	uuid "github.com/satori/go.uuid"
)

type Config struct {
	// Mandatory Setting, must be set
	PeerID  string `description:"the ipfs peer id of this node"  long:"peer-id"`
	PrivKey string `description:"the ipfs priv key of this node" long:"priv-key"`

	// Optional Settings
	BootstrapPeers []string `description:"a list of the multiaddress of bootstrap peers, will be used in addition to the ipfs default peers if enabled" long:"bootstrap-peers"`
	SwarmPort      int      `description:"ipfs swarm port"                                                                                              long:"swarm-port"`

	// Without this there would be no way to isolate an environment if needed and process a given chains data (e.g. for dev)
	SwarmKeyOverride string `description:"optional swarm key override, the default behaviour is to use the datanode's chain id'" long:"swarm-key-override"`

	HistoryRetentionBlockSpan int64 `description:"the block span of history, from the most recent history segment, that should be retained" long:"history-retention-block-span"`
}

func NewDefaultConfig() Config {
	seed := uuid.NewV4().String()
	identity, err := GenerateIdentityFromSeed([]byte(seed))
	if err != nil {
		panic("failed to generate default ipfs identity")
	}

	return Config{
		PeerID:  identity.PeerID,
		PrivKey: identity.PrivKey,

		BootstrapPeers: []string{},

		SwarmPort: 4001,

		HistoryRetentionBlockSpan: 604800, // One week of history at 1s per block
	}
}

func (c Config) GetSwarmKeySeed(log *logging.Logger, chainID string) string {
	swarmKeySeed := chainID
	if len(c.SwarmKeyOverride) > 0 {
		swarmKeySeed = c.SwarmKeyOverride
		log.Info("Using swarm key override as the swarm key seed", logging.String("swarm key seed", c.SwarmKeyOverride))
	} else {
		log.Info("Using chain id as the swarm key seed", logging.String("swarm key seed", c.SwarmKeyOverride))
	}
	return swarmKeySeed
}

func GenerateIdentityFromSeed(seed []byte) (config.Identity, error) {
	ident := config.Identity{}

	var sk crypto.PrivKey
	var pk crypto.PubKey

	// Everything > 32 bytes is ignored in GenerateEd25519Key so do a little pre hashing
	seedHash := sha256.Sum256(seed)

	priv, pub, err := crypto.GenerateEd25519Key(bytes.NewReader(seedHash[:]))
	if err != nil {
		return ident, err
	}

	sk = priv
	pk = pub

	skbytes, err := crypto.MarshalPrivateKey(sk)
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)

	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return ident, err
	}
	ident.PeerID = id.Pretty()
	return ident, nil
}
