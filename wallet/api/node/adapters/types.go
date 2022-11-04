package adapters

type Statistics struct {
	BlockHash   string
	BlockHeight uint64
	ChainID     string
	VegaTime    string
}

type LastBlock struct {
	ChainID                 string
	BlockHeight             uint64
	BlockHash               string
	ProofOfWorkHashFunction string
	ProofOfWorkDifficulty   uint32
}
