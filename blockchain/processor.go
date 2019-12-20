package blockchain

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// ProcessorService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/processor_service_mock.go -package mocks code.vegaprotocol.io/vega/blockchain ProcessorService
type ProcessorService interface {
	SubmitOrder(order *types.Order) error
	CancelOrder(order *types.Order) error
	AmendOrder(order *types.OrderAmendment) error
	NotifyTraderAccount(notify *types.NotifyTraderAccount) error
	Withdraw(*types.Withdraw) error
}

// Processor handle processing of all transaction sent through the node
type Processor struct {
	log *logging.Logger
	Config
	blockchainService ProcessorService
	seenPayloads      map[string]byte
}

// NewProcessor instantiates a new transactions processor
func NewProcessor(log *logging.Logger, config Config, blockchainService ProcessorService) *Processor {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Processor{
		log:               log,
		Config:            config,
		blockchainService: blockchainService,
		seenPayloads:      map[string]byte{},
	}
}

// ReloadConf update the internal configuration of the processor
func (p *Processor) ReloadConf(cfg Config) {
	p.log.Info("reloading configuration")
	if p.log.GetLevel() != cfg.Level.Get() {
		p.log.Info("updating log level",
			logging.String("old", p.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		p.log.SetLevel(cfg.Level.Get())
	}

	p.Config = cfg
}

func (p *Processor) getOrder(payload []byte) (*types.Order, error) {
	order := &types.Order{}
	err := proto.Unmarshal(payload, order)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (p *Processor) getOrderAmendment(payload []byte) (*types.OrderAmendment, error) {
	amendment := &types.OrderAmendment{}
	err := proto.Unmarshal(payload, amendment)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return amendment, nil
}

func (p *Processor) getNotifyTraderAccount(payload []byte) (*types.NotifyTraderAccount, error) {
	notif := &types.NotifyTraderAccount{}
	err := proto.Unmarshal(payload, notif)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return notif, nil
}

func (p *Processor) getWithdraw(payload []byte) (*types.Withdraw, error) {
	w := &types.Withdraw{}
	err := proto.Unmarshal(payload, w)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding order to proto")
	}
	return w, nil
}

// Validate performs all validation on an incoming transaction payload.
func (p *Processor) Validate(payload []byte) error {
	// Pre-validate (safety check)
	if seen, err := p.hasSeen(payload); seen {
		return errors.Wrap(err, "error during hasSeen (validate)")
	}
	// Attempt to decode transaction payload
	_, cmd, err := txDecode(payload)
	if err != nil {
		return errors.Wrap(err, "error decoding payload")
	}
	// Ensure valid VEGA app command
	switch cmd {
	case
		SubmitOrderCommand,
		CancelOrderCommand,
		AmendOrderCommand,
		NotifyTraderAccountCommand,
		WithdrawCommand:
		// Add future valid VEGA commands here
		return nil
	}
	return errors.New("unknown command when validating payload")

	// todo: Validation required here using blockchain service (gitlab.com/vega-protocol/trading-core/issues/177)
	//p.blockchainService.ValidateOrder()
}

// Process performs validation and then sends the command and data to
// the underlying blockchain service handlers e.g. submit order, etc.
func (p *Processor) Process(payload []byte) error {
	// Pre-validate (safety check)
	if seen, err := p.hasSeen(payload); seen {
		return errors.Wrap(err, "error during hasSeen (process)")
	}

	// Add to map of seen payloads, hashes only exist in here if they are processed.
	payloadHash, err := p.payloadHash(payload)
	if err != nil {
		return errors.Wrap(err, "error obtaining payload hash")
	}
	p.seenPayloads[*payloadHash] = 0xF

	// Attempt to decode transaction payload
	data, cmd, err := txDecode(payload)
	if err != nil {
		return errors.Wrap(err, "error decoding payload")
	}
	// Process known command types
	switch cmd {
	case SubmitOrderCommand:
		order, err := p.getOrder(data)
		if err != nil {
			return err
		}
		err = p.blockchainService.SubmitOrder(order)
		if err != nil {
			return err
		}
	case CancelOrderCommand:
		order, err := p.getOrder(data)
		if err != nil {
			return err
		}
		err = p.blockchainService.CancelOrder(order)
		if err != nil {
			return err
		}
	case AmendOrderCommand:
		order, err := p.getOrderAmendment(data)
		if err != nil {
			return err
		}
		err = p.blockchainService.AmendOrder(order)
		if err != nil {
			return err
		}
	case NotifyTraderAccountCommand:
		notify, err := p.getNotifyTraderAccount(data)
		if err != nil {
			return err
		}
		err = p.blockchainService.NotifyTraderAccount(notify)
		if err != nil {
			return err
		}
	case WithdrawCommand:
		w, err := p.getWithdraw(data)
		if err != nil {
			return err
		}
		err = p.blockchainService.Withdraw(w)
		if err != nil {
			return err
		}
	default:
		p.log.Warn("Unknown command received", logging.String("command", string(cmd)))
		return fmt.Errorf("unknown command received: %s", cmd)
	}
	return nil
}

// hasSeen helper performs duplicate checking on an incoming transaction payload.
func (p *Processor) hasSeen(payload []byte) (bool, error) {
	// All vega transactions are prefixed with a unique hash, using
	// this means we do not have to re-compute each time for seen keys
	payloadHash, err := p.payloadHash(payload)
	if err != nil {
		return true, err
	}
	// Safety checks at business level to ensure duplicate transaction
	// payloads do not pass through to application core
	if exists, err := p.payloadExists(payloadHash); exists {
		return true, err
	}

	return false, nil
}

// payloadHash attempts to extract the unique hash at the start of all vega transactions.
// This unique hash is required to make all payloads unique. We return an error if we cannot
// extract this from the transaction payload or if we think it's malformed.
func (p *Processor) payloadHash(payload []byte) (*string, error) {
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
func (p *Processor) payloadExists(payloadHash *string) (bool, error) {
	if _, exists := p.seenPayloads[*payloadHash]; exists {
		p.log.Warn("Transaction payload exists", logging.String("payload-hash", *payloadHash))
		err := fmt.Errorf("txn payload exists: %s", *payloadHash)
		return true, err
	}
	return false, nil
}

// ResetSeenPayloads is used to reset the map containing the list of keys for payloads
// seen in the current batch, seenPayloads is a safety check for dupes per batch.
func (p *Processor) ResetSeenPayloads() {
	p.seenPayloads = map[string]byte{}
}

// txDecode is takes the raw payload bytes and decodes the contents using a pre-defined
// strategy, we have a simple and efficient encoding at present. A partner encode function
// can be found in the blockchain client.
func txDecode(input []byte) (proto []byte, cmd Command, err error) {
	// Input is typically the bytes that arrive in raw format after consensus is reached.
	// Split the transaction dropping the unification bytes (uuid&pipe)
	var value []byte
	var cmdByte byte
	if len(input) > 37 {
		// obtain command from byte slice (0 indexed)
		cmdByte = input[36]
		// remaining bytes are payload
		value = input[37:]
	} else {
		return nil, 0, errors.New("payload size is incorrect, should be > 38 bytes")
	}
	return value, Command(cmdByte), nil
}
