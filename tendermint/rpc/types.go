package rpc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
	"vega/tendermint/types"
)

// Block represents a block in the Tendermint blockchain.
type Block struct {
	Data       *Data     `json:"data"`
	Evidence   *Evidence `json:"evidence"`
	Header     *Header   `json:"header"`
	PrevCommit *Commit   `json:"last_commit"`
}

// BlockID is comprised of the Simple Tree hash of the block header encoded as a
// list of KVPairs, along with the PartSetHeader.
type BlockID struct {
	HeaderHash ByteSlice      `json:"hash"`
	Parts      *PartSetHeader `json:"parts"`
}

// BlockInfo represents the data and metadata for an individual block in the
// Tendermint blockchain.
type BlockInfo struct {
	Block Block     `json:"block"`
	Meta  BlockMeta `json:"block_meta"`
}

// BlockMeta represents metadata about a block in the Tendermint blockchain.
type BlockMeta struct {
	BlockID BlockID `json:"block_id"` // the block hash and partsethash
	Header  *Header `json:"header"`   // The block's Header
}

// BlockSizeParams specifies the limits on the block size.
type BlockSizeParams struct {
	MaxBytes int   `json:"max_bytes"` // NOTE: must not be 0 nor greater than 100MB.
	MaxGas   int64 `json:"max_gas"`
	MaxTxs   int   `json:"max_txs"`
}

// ByteSlice matches how Tendermint encodes certain byte slices in JSON.
type ByteSlice []byte

// MarshalJSON hex-encodes the bytes and quotes it as a JSON string.
func (b ByteSlice) MarshalJSON() ([]byte, error) {
	buf := make([]byte, (len(b)*2)+2)
	buf[0] = '"'
	buf[len(buf)-1] = '"'
	hex.Encode(buf[1:len(buf)-1], b)
	return buf, nil
}

// UnmarshalJSON unquotes the JSON string and hex-decodes it.
func (b *ByteSlice) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' || len(data)%2 != 0 {
		return fmt.Errorf("rpc: invalid ByteSlice JSON encoding: %s", data)
	}
	buf := make([]byte, (len(data)-2)/2)
	if _, err := hex.Decode(buf, data[1:len(data)-1]); err != nil {
		return err
	}
	*b = buf
	return nil
}

// CheckTxResult includes some of the metadata from the CheckTx ABCI call for a
// newly added transaction.
type CheckTxResult struct {
	// A non-zero Code value represents an error. The meaning of non-zero codes
	// is specific to the given ABCI app that is being used.
	Code uint32    `json:"code"`
	Data ByteSlice `json:"data"`
	Hash ByteSlice `json:"hash"`
	Log  string    `json:"log"`
}

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.

// Commit contains a set of votes that were made by the validator set to reach
// consensus on a block.
type Commit struct {
	// NOTE: The Precommits are in order of address to preserve the bonded ValidatorSet order.
	// Any peer with a block can gossip precommits by index with a peer without recalculating the
	// active ValidatorSet.
	BlockID    BlockID `json:"block_id"`
	Precommits []*Vote `json:"precommits"`
}

// CommitInfo represents
type CommitInfo struct {
	Canonical bool
}

// ConsensusParams defines the key parameters that determine the validity of
// blocks in the Tendermint blockchain.
type ConsensusParams struct {
	BlockSize BlockSizeParams `json:"block_size_params"`
	Evidence  EvidenceParams  `json:"evidence_params"`
	Gossip    GossipParams    `json:"block_gossip_params"`
	TxSize    TxSizeParams    `json:"tx_size_params"`
}

// Data contains the list of transactions that are to be processed, i.e. the
// ones that will be applied by the state @ block.Header.Height + 1.
//
// NOTE: Not all transactions here are valid. We're just agreeing on the order
// first. This means that block.Header.AppHash does not include these.
type Data struct {
	Transactions [][]byte `json:"txs"`
}

// DuplicateVoteEvidence contains evidence that a validator signed two
// conflicting votes.
type DuplicateVoteEvidence struct {
	PubKey TaggedByteSlice
	VoteA  *Vote
	VoteB  *Vote
}

// Evidence contains any evidence of malicious wrong-doing by validators.
type Evidence struct {
	List []*TaggedValue `json:"evidence"`
}

// EvidenceParams is used to determine how evidence of malfeasance is handled.
type EvidenceParams struct {
	MaxAge int64 `json:"max_age"` // Only accept new evidence more recent than this.
}

// Genesis specifies the initial conditions of the Tendermint blockchain.
type Genesis struct {
	AppHash         ByteSlice          `json:"app_hash"`
	AppStateJSON    json.RawMessage    `json:"app_state"`
	ChainID         string             `json:"chain_id"`
	ConsensusParams *ConsensusParams   `json:"consensus_params"`
	GenesisTime     time.Time          `json:"genesis_time"`
	Validators      []GenesisValidator `json:"validators"`
}

// GenesisValidator represents an initial Tendermint validator.
type GenesisValidator struct {
	Name        string          `json:"name"`
	PubKey      TaggedByteSlice `json:"pub_key"`
	VotingPower int64           `json:"power"`
}

// GossipParams defines the parameters relating to how blocks are gossiped.
type GossipParams struct {
	BlockPartSizeBytes int `json:"block_part_size_bytes"` // NOTE: must not be 0.
}

// Header defines the structure of a Tendermint block header.
type Header struct {
	ChainID string    `json:"chain_id"`
	Height  int64     `json:"height"`
	Time    time.Time `json:"time"`
	NumTxs  int64     `json:"num_txs"`

	// prev block info
	PrevBlockID BlockID `json:"last_block_id"`
	TotalTxs    int64   `json:"total_txs"`

	// hashes of block data
	PrevCommitHash ByteSlice `json:"last_commit_hash"` // commit from validators from the last block
	DataHash       ByteSlice `json:"data_hash"`        // transactions

	// hashes from the app output from the prev block
	ValidatorsHash  ByteSlice `json:"validators_hash"`   // validators for the current block
	ConsensusHash   ByteSlice `json:"consensus_hash"`    // consensus params for current block
	AppHash         ByteSlice `json:"app_hash"`          // state after txs from the previous block
	PrevResultsHash ByteSlice `json:"last_results_hash"` // root hash of all results from the txs from the previous block

	// consensus info
	EvidenceHash ByteSlice `json:"evidence_hash"` // evidence included in the block
}

// NetInfo represents current network info.
type NetInfo struct {
	Listening bool     `json:"listening"`
	Listeners []string `json:"listeners"`
	nPeersRaw string   `json:"n_peers"`
	NPeers    int
	Peers     []Peer   `json:"peers"`
}

// A peer
type Peer struct {
	NodeInfo         `json:"node_info"`
	IsOutbound       bool             `json:"is_outbound"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
}

// ConnectionStatus represents the status of the current connection and internal monitors.
type ConnectionStatus struct {
	Duration    string
	SendMonitor MonitorStatus
	RecvMonitor MonitorStatus
	Channels    []ChannelStatus
}

// ChannelStatus represents the status of a current channel
type ChannelStatus struct {
	ID                byte
	SendQueueCapacity int
	SendQueueSize     int
	Priority          int
	RecentlySent      string
}

// MonitorStatus represents the current Monitor status. All transfer rates are in bytes
// per second rounded to the nearest byte.
type MonitorStatus struct {
	Active   bool   // Flag indicating an active transfer
	Start    string // Transfer start time
	Duration string // Time period covered by the statistics
	Idle     string // Time since the last transfer of at least 1 byte
	Bytes    string // Total number of bytes transferred
	Samples  string // Total number of samples taken
	InstRate string // Instantaneous transfer rate
	CurRate  string // Current transfer rate (EMA of InstRate)
	AvgRate  string // Average transfer rate (Bytes / Duration)
	PeakRate string // Maximum instantaneous transfer rate
	BytesRem string // Number of bytes remaining in the transfer
	TimeRem  string // Estimated time to completion
	Progress uint32 // Overall transfer progress
}

// NodeInfo is the basic node information exchanged
// between two peers during the Tendermint P2P handshake.
type NodeInfo struct {
	// Authenticate
	ID         string `json:"id"`          // authenticated identifier
	ListenAddr string `json:"listen_addr"` // accepting incoming

	// Check compatibility.
	// Channels are HexBytes so easier to read as JSON
	Network  string    `json:"network"`  // network/chain ID
	Version  string    `json:"version"`  // major.minor.revision
	Channels ByteSlice `json:"channels"` // channels this node knows about

	// ASCIIText fields
	Moniker string   `json:"moniker"` // arbitrary moniker
	Other   []string `json:"other"`   // other application specific data
}

// PartSetHeader defines the total number of pieces in a PartSet and the Merkle
// root hash of those pieces. PartSet is simply the way that Tendermint chunks
// up large data into smaller pieces for transmission across the network.
type PartSetHeader struct {
	Hash  ByteSlice `json:"hash"`
	Total int       `json:"total"`
}

// Status of Tendermint including; node info, pubkey, latest block
// hash, app hash, block height and time.
type Status struct {
	NodeInfo      NodeInfo  `json:"node_info"`
	SyncInfo      SyncInfo  `json:"sync_info"`
	ValidatorInfo Validator `json:"validator_info"`
}

// Info about the node's syncing state
type SyncInfo struct {
	LatestBlockHash   ByteSlice `json:"latest_block_hash"`
	LatestAppHash     ByteSlice `json:"latest_app_hash"`
	LatestBlockHeight int64     `json:"latest_block_height"`
	LatestBlockTime   time.Time `json:"latest_block_time"`
	CatchingUp        bool      `json:"catching_up"`
}

// TaggedByteSlice represents a byte slice encoded using Tendermint's custom
// Amino serialisation format.
//
// The value is tagged with a Type corresponding to the internal "disfix" ID
// (disambiguation bytes + prefix bytes) used by Amino to distinguish different
// types.
type TaggedByteSlice struct {
	Type  ByteSlice `json:"type"`
	Value []byte    `json:"value"`
}

// TaggedValue represents a value encoded using Tendermint's custom Amino
// serialisation format. It is tagged with a Type similar to TaggedByteSlice.
type TaggedValue struct {
	Type  ByteSlice       `json:"type"`
	Value json.RawMessage `json:"value"`
}

// As attempts to decode the underlying TaggedValue and stores the result in the
// value pointed to by v.
func (t *TaggedValue) As(v interface{}) error {
	return json.Unmarshal(t.Value, v)
}

// TxSizeParams specifies the limits relating to individual transactions.
type TxSizeParams struct {
	MaxBytes int   `json:"max_bytes"`
	MaxGas   int64 `json:"max_gas"`
}

// Transaction represents the result of querying for a transaction.
type Transaction struct {
	Hash     ByteSlice               `json:"hash"`
	Height   int64                   `json:"height"`
	Index    uint32                  `json:"index"`
	TxResult types.DeliverTxResponse `json:"tx_result"`
	Tx       ByteSlice               `json:"tx"`
	Proof    TxProof                 `json:"proof,omitempty"`
}

// TxProof represents a Merkle proof of the presence of a transaction in the Merkle tree.
type TxProof struct {
	Index, Total int
	RootHash     ByteSlice
	Data         ByteSlice
	Proof        SimpleProof
}

// SimpleProof represents a simple merkle proof.
type SimpleProof struct {
	Aunts [][]byte `json:"aunts"` // Hashes from leaf's sibling to a root's child.
}

type TransactionList struct {
	Count        int            `json:"total_count"`
	Transactions []*Transaction `json:"txs"`
}

// Validator represents a Tendermint validator node. Its structure is slightly
// different to the GenesisValidator which is only used for the initial set of
// validators.
type Validator struct {
	Accum       int64           `json:"accum"`
	Address     TaggedByteSlice `json:"address"`
	PubKey      TaggedByteSlice `json:"pub_key"`
	VotingPower int64           `json:"voting_power"`
}

// ValidatorSet represents the set of validators at the given block height.
type ValidatorSet struct {
	Height     int64        `json:"block_height"`
	Validators []*Validator `json:"validators"`
}

// Vote represents a prevote, precommit, or commit vote from validators.
type Vote struct {
	BlockID          BlockID         `json:"block_id"`
	Height           int64           `json:"height"`
	Round            int             `json:"round"`
	Signature        TaggedByteSlice `json:"signature"`
	Timestamp        time.Time       `json:"timestamp"`
	Type             byte            `json:"type"`
	ValidatorAddress ByteSlice       `json:"validator_address"`
	ValidatorIndex   int             `json:"validator_index"`
}
