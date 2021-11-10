package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"code.vegaprotocol.io/vega/processor"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

const (
	endpoint          = "http://%v/block?height=%d"
	endpointLastBlock = "http://%v/block"
)

var (
	opts = struct {
		from      uint64
		to        uint64
		tmAddress string
	}{}
)

type Payload struct {
	Result struct {
		BlockID struct {
			Hash string `json:"hash"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				Height string `json:"height"`
			} `json:"header"`
			Data struct {
				Txs []string `json:"txs"`
			} `json:"data"`
		} `json:"block"`
	} `json:"result"`
}

type Transaction struct {
	Command   json.RawMessage
	Signature json.RawMessage
	PubKey    string
}

func init() {
	flag.Uint64Var(&opts.from, "from", 0, "the first block to pull from the tendermint chain")
	flag.Uint64Var(&opts.to, "to", 0, "the last block to pull from the tendermint chain, default to current height if set to 0")
	flag.StringVar(&opts.tmAddress, "tendermint", "142.93.46.33:26657", "tendermint node address")
}

func getLastBlockHeight() uint64 {
	resp, err := http.Get(fmt.Sprintf(endpointLastBlock, opts.tmAddress))
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	p := Payload{}
	if err := json.Unmarshal(body, &p); err != nil {
		panic(err)
	}

	u, err := strconv.ParseUint(p.Result.Block.Header.Height, 10, 64)
	if err != nil {
		panic(err)
	}

	return u
}

func getBlock(height uint64) *Payload {
	resp, err := http.Get(fmtEndpoint(height))
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	p := Payload{}
	if err := json.Unmarshal(body, &p); err != nil {
		panic(err)
	}

	return &p
}

func main() {
	flag.Parse()

	if opts.from > opts.to && opts.to != 0 {
		fmt.Printf("error: -from cannot be hight than -to")
		os.Exit(1)
	}

	if opts.to == 0 {
		opts.to = getLastBlockHeight()
	}

	for i := opts.from; i < opts.to; i++ {
		p := getBlock(i)

		fmt.Printf("%v - %d\n", p.Result.BlockID.Hash, i)
		for _, v := range p.Result.Block.Data.Txs {
			buf, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				panic(err)
			}

			tx, err := processor.DecodeTxV2(buf)
			if err != nil {
				panic(err)
			}

			m := jsonpb.Marshaler{
				Indent:       "  ",
				EmitDefaults: true,
			}
			txMarshalled, err := m.MarshalToString(tx.RawTx().Signature)
			if err != nil {
				panic(err)
			}
			commandMarshalled, err := m.MarshalToString(tx.GetCmd().(proto.Message))
			if err != nil {
				panic(err)
			}
			finalTx := Transaction{
				Command:   []byte(commandMarshalled),
				Signature: []byte(txMarshalled),
				PubKey:    tx.PubKeyHex(),
			}

			buf2, err := json.MarshalIndent(&finalTx, "", " ")
			if err != nil {
				panic(err)
			}

			fmt.Printf("%v - %v\n", tx.Command().String(), string(buf2))
		}
	}
}

func fmtEndpoint(i uint64) string {
	return fmt.Sprintf(endpoint, opts.tmAddress, i)
}
