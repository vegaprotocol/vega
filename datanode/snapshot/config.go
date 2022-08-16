package snapshot

type Config struct {
	StartHeight int64  `long:"load-from-block-height" description:"Start the node by loading the snapshot taken at the given block-height. -1 for last snapshot, 0 for no reload (default: 0)"` // -1 for last snapshot, 0 for no reload
	ChainId     string `long:"chain-id" description:"The chain id of the snapshot to restore, not required if the snapshot folder contains only one snapshot for the given height"`

	BlockInterval int64 `long:"block-interval" description:"the block interval between create=ion of snapshots, 0 if snapshot create should be disabled (default: 0)"`
	HistorySize   int   `long:"history-size" description:"the maximum number of snapshots to keep locally"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		ChainId:       "",
		StartHeight:   0,
		BlockInterval: 20000, // Roughly 6 hours at 1 block per second
		HistorySize:   5,
	}
}
