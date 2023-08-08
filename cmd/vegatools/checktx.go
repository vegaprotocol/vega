package tools

import (
	"encoding/base64"
	"fmt"

	"github.com/sirupsen/logrus"

	"code.vegaprotocol.io/vega/core/config"
	"code.vegaprotocol.io/vega/libs/proto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type checkTxCmd struct {
	config.OutputFlag
	EncodedTransaction string `description:"The encoded transaction string to compare with vega's own encoding" long:"tx" short:"t" required:"true"`
}

func (opts *checkTxCmd) Execute(_ []string) error {
	logrus.Infof("checking transaction...")
	unmarshalledTransaction, err := decodeAndUnmarshalTransaction(opts.EncodedTransaction)
	if err != nil {
		return fmt.Errorf("error occurred when attempting to unmarshal the encoded transaction data.\nerr: %w", err)
	}

	logrus.Infof("reencoding the tranaction...")
	reEncodedTransaction, err := marshalAndEncodeTransaction(unmarshalledTransaction)

	if reEncodedTransaction != opts.EncodedTransaction {
		logrus.Errorf("transactions not equal!")
		return fmt.Errorf("the reencoded transaction was not equal. \nOriginal: %s\nVegaEncoded: %s", opts.EncodedTransaction, reEncodedTransaction)
	}

	logrus.Infof("transactions equal!")
	return nil
}

func decodeAndUnmarshalTransaction(encodedTransaction string) (*commandspb.Transaction, error) {
	unmarshalledTransaction := &commandspb.Transaction{}
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedTransaction)
	if err != nil {
		return unmarshalledTransaction, fmt.Errorf("error occurred when decoding the encoded data.\nerr: %w", err)
	}

	if err := proto.Unmarshal(decodedBytes, unmarshalledTransaction); err != nil {
		return unmarshalledTransaction, fmt.Errorf("unable to unmarshal the decided bytes to a transaction: %w", err)
	}

	logrus.Infof("successfully decoded and unmarshalled the transaction")

	return unmarshalledTransaction, nil
}

func marshalAndEncodeTransaction(transaction *commandspb.Transaction) (string, error) {
	pb, err := proto.Marshal(transaction)
	if err != nil {
		return "", fmt.Errorf("error when attempting to marshal Transaction struct back to a proto.\nerr: %w", err)
	}

	reEncodedTransaction := base64.StdEncoding.EncodeToString(pb)
	logrus.Infof("successfully marshalled to proto and encoded")
	return reEncodedTransaction, nil
}
