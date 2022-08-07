package crypto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"math"
)

const (
	Sha3     = "sha3_24_rounds"
	maxNonce = math.MaxInt64
)

var prefix = []byte("Vega_SPAM_PoW")

// PoW calculates proof of work given block hash, transaction hash, target difficulty and a hash function.
// returns the nonce, the hash and th error if any.
func PoW(blockHash string, txID string, difficulty uint, hashFunction string) (uint64, []byte, error) {
	var h []byte
	var err error
	nonce := uint64(0)

	if difficulty > 256 {
		return nonce, h, errors.New("invalid difficulty")
	}

	if len(txID) < 1 {
		return nonce, h, errors.New("transaction ID cannot be empty")
	}

	if len(blockHash) != 64 {
		return nonce, h, errors.New("incorrect block hash")
	}

	for nonce < maxNonce {
		data := prepareData(blockHash, txID, nonce)
		h, err = hash(data, hashFunction)
		if err != nil {
			return nonce, h, err
		}

		if CountZeros(h) >= byte(difficulty) {
			break
		} else {
			nonce++
		}
	}

	return nonce, h[:], nil
}

// Verify checks that the hash with the given nonce results in the target difficulty.
func Verify(blockHash string, tid string, nonce uint64, hashFuncion string, difficulty uint) (bool, byte) {
	if difficulty > 256 {
		return false, 0
	}

	if len(tid) < 1 {
		return false, 0
	}

	if len(blockHash) != 64 {
		return false, 0
	}

	data := prepareData(blockHash, tid, nonce)
	h, err := hash(data, hashFuncion)
	if err != nil {
		return false, 0
	}
	hDiff := CountZeros(h)
	return hDiff >= byte(difficulty), hDiff
}

func CountZeros(d []byte) byte {
	var ret byte
	for _, x := range d {
		if x == 0 {
			ret += 8
		} else {
			if x&128 != 0x00 {
				break
			}
			if x&64 != 0x00 {
				ret++
				break
			}
			if x&32 != 0x00 {
				ret += 2
				break
			}
			if x&16 != 0x00 {
				ret += 3
				break
			}
			if x&8 != 0x00 {
				ret += 4
				break
			}
			if x&4 != 0x00 {
				ret += 5
				break
			}
			if x&2 != 0x00 {
				ret += 6
				break
			}
			if x&1 != 0x00 {
				ret += 7
			}
			break
		}
	}
	return ret
}

func prepareData(blockHash string, txID string, nonce uint64) []byte {
	data := bytes.Join(
		[][]byte{
			prefix,
			[]byte(blockHash),
			[]byte(txID),
			IntToHex(nonce),
		},
		[]byte{},
	)

	return data
}

func hash(data []byte, hashFunction string) ([]byte, error) {
	if hashFunction == Sha3 {
		return Hash(data), nil
	}
	return nil, errors.New("unknown hash function")
}

func IntToHex(num uint64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}
