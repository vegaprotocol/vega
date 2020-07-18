package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	ErrCommandKindUnknown = errors.New("unknown command kind when validating payload")
)

type Processor interface {
	Process(ctx context.Context, payload []byte, pubkey []byte, cmd Command) error
	ValidateSigned(key, payload []byte, cmd Command) error
}

type codec struct {
	Config
	log          *logging.Logger
	p            Processor
	seenPayloads map[string]struct{}
}

func NewCodec(log *logging.Logger, conf Config, p Processor) *codec {
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())
	return &codec{
		Config:       conf,
		log:          log,
		p:            p,
		seenPayloads: map[string]struct{}{},
	}
}

// ReloadConf update the internal configuration of the processor
func (c *codec) ReloadConf(cfg Config) {
	c.log.Info("reloading configuration")
	if c.log.GetLevel() != cfg.Level.Get() {
		c.log.Info("updating log level",
			logging.String("old", c.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		c.log.SetLevel(cfg.Level.Get())
	}

	c.Config = cfg
}

// Validate performs all validation on an incoming transaction payload.
func (c *codec) Validate(payload []byte) error {
	// Pre-validate (safety check)
	if seen, err := c.hasSeen(payload); seen {
		return errors.Wrap(err, "error during hasSeen (validate)")
	}

	return c.validateSigned(payload)
}

func (c *codec) Process(payload []byte) error {
	// Pre-validate (safety check)
	if seen, err := c.hasSeen(payload); seen {
		return errors.Wrap(err, "error during hasSeen (process)")
	}

	// Add to map of seen payloads, hashes only exist in here if they are processed.
	payloadHash, err := c.payloadHash(payload)
	if err != nil {
		return errors.Wrap(err, "error obtaining payload hash")
	}
	c.seenPayloads[*payloadHash] = struct{}{}

	hexPayloadHash := hex.EncodeToString([]byte(*payloadHash))
	// get the block context, add transaction hash as trace ID
	ctx := contextutil.WithTraceID(context.Background(), hexPayloadHash)

	// first unmarshal the bundle
	bundle := &types.SignedBundle{}
	if err := proto.Unmarshal(payload, bundle); err != nil {
		c.log.Error("unable to unmarshal signed bundle", logging.Error(err))
		return err
	}

	tx := &types.Transaction{}
	err = proto.Unmarshal(bundle.Tx, tx)
	if err != nil {
		c.log.Error("unable to unmarshal Transction", logging.Error(err))
	}

	cmdData, cmd, err := txDecode(tx.InputData)
	if err != nil {
		c.log.Error("could not process transaction, error decoding",
			logging.Error(err))
		return err
	}

	// FIXME(): signature needs to be forwarded as well
	// Actually process the transaction
	return c.p.Process(ctx, cmdData, tx.GetPubKey(), cmd)
}

func (c *codec) validateSigned(payload []byte) error {
	// first unmarshal the bundle
	bundle := &types.SignedBundle{}
	err := proto.Unmarshal(payload, bundle)
	if err != nil {
		c.log.Error("unable to unmarshal signed bundle", logging.Error(err))
		return err
	}

	tx := &types.Transaction{}
	err = proto.Unmarshal(bundle.Tx, tx)
	if err != nil {
		c.log.Error("unable to unmarshal transaction from signed bundle",
			logging.Error(err))
		return err
	}

	cmdData, cmd, err := txDecode(tx.InputData)
	if err != nil {
		return errors.Wrap(err, "error decoding payload")
	}

	// FIXME(): for now we just not verify 2 command which are
	// not require to be signed. This will need to be removed once we have
	// only signed commadn
	if cmd != WithdrawCommand && cmd != NotifyTraderAccountCommand {
		// verify the signature
		if err := verifyBundle(c.log, tx, bundle); err != nil {
			c.log.Error("error verifying bundle", logging.Error(err))
			return err
		}
	}

	if _, ok := commandName[cmd]; !ok {
		return errors.New("unknown command when validating payload")
	}
	return c.p.ValidateSigned(tx.GetPubKey(), cmdData, cmd)
}

// hasSeen helper performs duplicate checking on an incoming transaction payload.
func (c *codec) hasSeen(payload []byte) (bool, error) {
	// All vega transactions are prefixed with a unique hash, using
	// this means we do not have to re-compute each time for seen keys
	payloadHash, err := c.payloadHash(payload)
	if err != nil {
		return true, err
	}
	// Safety checks at business level to ensure duplicate transaction
	// payloads do not pass through to application core
	return c.payloadExists(payloadHash)
}

// payloadHash attempts to extract the unique hash at the start of all vega transactions.
// This unique hash is required to make all payloads unique. We return an error if we cannot
// extract this from the transaction payload or if we think it's malformed.
func (c *codec) payloadHash(payload []byte) (*string, error) {
	if len(payload) < 36 {
		return nil, errors.New("invalid length payload, must be greater than 37 bytes")
	}
	hash := string(payload[0:36])
	return &hash, nil
}

// payloadExists checks to see if a payload has been seen before in this batch
// recommended by tendermint team that an abci application has additional checking
// just like this to ensure duplicate transaction payloads do not pass through
// to the application core.
func (c *codec) payloadExists(payloadHash *string) (bool, error) {
	if _, exists := c.seenPayloads[*payloadHash]; exists {
		c.log.Warn("Transaction payload exists", logging.String("payload-hash", *payloadHash))
		return true, fmt.Errorf("txn payload exists: %s", *payloadHash)
	}
	return false, nil
}

// txDecode is takes the raw payload bytes and decodes the contents using a pre-defined
// strategy, we have a simple and efficient encoding at present. A partner encode function
// can be found in the blockchain client.
func txDecode(input []byte) ([]byte, Command, error) {
	// Input is typically the bytes that arrive in raw format after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	if len(input) > 37 {
		// obtain command from byte slice (0 indexed)
		// remaining bytes are payload
		return input[37:], Command(input[36]), nil
	}
	return nil, 0, errors.New("payload size is incorrect, should be > 38 bytes")
}

func txEncode(input []byte, cmd Command) ([]byte, error) {
	prefix := uuid.NewV4().String()
	prefixBytes := []byte(prefix)
	commandInput := append([]byte{byte(cmd)}, input...)
	return append(prefixBytes, commandInput...), nil
}

// ResetSeenPayloads is used to reset the map containing the list of keys for payloads
// seen in the current batch, seenPayloads is a safety check for dupes per batch.
func (c *codec) ResetSeenPayloads() {
	c.seenPayloads = map[string]struct{}{}
}
