package proto

import (
	"errors"
	fmt "fmt"
)

var (
	ErrUnsupportedChainEvent = errors.New("unsupported chain event")
)

func (c *ChainEvent) PrepareToSign() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(c.TxID)...)
	out = append(out, []byte(fmt.Sprintf("%v", c.Nonce))...)

	var (
		next []byte
		err  error
	)
	switch evt := c.Event.(type) {
	case *ChainEvent_Erc20:
		next, err = evt.Erc20.PrepareToSign()
	case *ChainEvent_Builtin:
		next, err = evt.Builtin.PrepareToSign()
	default:
		err = ErrUnsupportedChainEvent
	}

	if err != nil {
		return nil, err
	}

	out = append(out, next...)
	return out, nil
}

func (c *BuiltinAssetEvent) PrepareToSign() ([]byte, error) {
	switch act := c.Action.(type) {
	case *BuiltinAssetEvent_Deposit:
		return act.Deposit.PrepareToSign()
	case *BuiltinAssetEvent_Withdrawal:
		return act.Withdrawal.PrepareToSign()
	default:
		return nil, ErrUnsupportedChainEvent
	}
}

func (c *BuiltinAssetDeposit) PrepareToSign() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(c.VegaAssetID)...)
	out = append(out, []byte(c.PartyID)...)
	out = append(out, []byte(fmt.Sprintf("%v", c.Amount))...)
	return out, nil
}

func (c *BuiltinAssetWithdrawal) PrepareToSign() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(c.VegaAssetID)...)
	out = append(out, []byte(c.PartyID)...)
	out = append(out, []byte(fmt.Sprintf("%v", c.Amount))...)
	return out, nil
}

func (c *ERC20Event) PrepareToSign() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(fmt.Sprintf("%v", c.Block))...)
	out = append(out, []byte(fmt.Sprintf("%v", c.Index))...)

	fmt.Printf("PREPARE To SIGN ERC20EVENT\n")
	var (
		next []byte
		err  error
	)
	switch act := c.Action.(type) {
	case *ERC20Event_AssetList:
		next, err = act.AssetList.PrepareToSign()
	case *ERC20Event_AssetDelist:
		next, err = act.AssetDelist.PrepareToSign()
	case *ERC20Event_Deposit:
		fmt.Printf("PREPARE IN DEPOSIT\n")
		next, err = act.Deposit.PrepareToSign()
	case *ERC20Event_Withdrawal:
		fmt.Printf("PREPARE IN WITHDRAWAL\n")
		next, err = act.Withdrawal.PrepareToSign()
	default:
		err = ErrUnsupportedChainEvent
	}

	if err != nil {
		return nil, err
	}

	out = append(out, next...)
	return out, nil
}

func (c *ERC20AssetList) PrepareToSign() ([]byte, error) {
	return []byte(c.VegaAssetID), nil
}

func (c *ERC20AssetDelist) PrepareToSign() ([]byte, error) {
	return []byte(c.VegaAssetID), nil
}

func (c *ERC20Deposit) PrepareToSign() ([]byte, error) {
	out := []byte{}
	out = append(out, []byte(c.VegaAssetID)...)
	out = append(out, []byte(c.SourceEthereumAddress)...)
	out = append(out, []byte(c.TargetPartyID)...)
	return out, nil
}

func (c *ERC20Withdrawal) PrepareToSign() ([]byte, error) {
	out := []byte{}
	fmt.Printf("PREPARE SIGN WITHDRAWAL\n")
	fmt.Printf("REFERENCE NONCE: %v\n", c.ReferenceNonce)
	fmt.Printf("REFERENCE NONCE: %v\n", c.ReferenceNonce)
	fmt.Printf("REFERENCE NONCE: %v\n", c.ReferenceNonce)
	out = append(out, []byte(c.VegaAssetID)...)
	out = append(out, []byte(c.TargetEthereumAddress)...)
	out = append(out, []byte(c.ReferenceNonce)...)
	return out, nil
}
