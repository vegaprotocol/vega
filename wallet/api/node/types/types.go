package types

import "fmt"

type TransactionError struct {
	ABCICode uint32
	Message  string
}

func (e TransactionError) Error() string {
	return fmt.Sprintf("%s (ABCI code %d)", e.Message, e.ABCICode)
}

func (e TransactionError) Is(target error) bool {
	_, ok := target.(TransactionError)
	return ok
}

type Statistics struct {
	BlockHash   string
	BlockHeight uint64
	ChainID     string
	VegaTime    string
}

type LastBlock struct {
	ChainID                         string
	BlockHeight                     uint64
	BlockHash                       string
	ProofOfWorkHashFunction         string
	ProofOfWorkDifficulty           uint32
	ProofOfWorkPastBlocks           uint32
	ProofOfWorkTxPerBlock           uint32
	ProofOfWorkIncreasingDifficulty bool
}
