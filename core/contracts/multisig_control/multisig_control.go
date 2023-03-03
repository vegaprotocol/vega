// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package multisig_control

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// MultisigControlMetaData contains all meta data concerning the MultisigControl contract.
var MultisigControlMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"NonceBurnt\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"new_signer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"SignerAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"old_signer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"SignerRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"new_threshold\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"ThresholdSet\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"new_signer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"add_signer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"burn_nonce\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_current_threshold\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_valid_signer_count\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"is_nonce_used\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"signer_address\",\"type\":\"address\"}],\"name\":\"is_valid_signer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"old_signer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"remove_signer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"new_threshold\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"set_threshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"signers\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"verify_signatures\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// MultisigControlABI is the input ABI used to generate the binding from.
// Deprecated: Use MultisigControlMetaData.ABI instead.
var MultisigControlABI = MultisigControlMetaData.ABI

// MultisigControl is an auto generated Go binding around an Ethereum contract.
type MultisigControl struct {
	MultisigControlCaller     // Read-only binding to the contract
	MultisigControlTransactor // Write-only binding to the contract
	MultisigControlFilterer   // Log filterer for contract events
}

// MultisigControlCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultisigControlCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultisigControlTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultisigControlTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultisigControlFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultisigControlFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultisigControlSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultisigControlSession struct {
	Contract     *MultisigControl  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultisigControlCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultisigControlCallerSession struct {
	Contract *MultisigControlCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// MultisigControlTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultisigControlTransactorSession struct {
	Contract     *MultisigControlTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// MultisigControlRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultisigControlRaw struct {
	Contract *MultisigControl // Generic contract binding to access the raw methods on
}

// MultisigControlCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultisigControlCallerRaw struct {
	Contract *MultisigControlCaller // Generic read-only contract binding to access the raw methods on
}

// MultisigControlTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultisigControlTransactorRaw struct {
	Contract *MultisigControlTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultisigControl creates a new instance of MultisigControl, bound to a specific deployed contract.
func NewMultisigControl(address common.Address, backend bind.ContractBackend) (*MultisigControl, error) {
	contract, err := bindMultisigControl(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultisigControl{MultisigControlCaller: MultisigControlCaller{contract: contract}, MultisigControlTransactor: MultisigControlTransactor{contract: contract}, MultisigControlFilterer: MultisigControlFilterer{contract: contract}}, nil
}

// NewMultisigControlCaller creates a new read-only instance of MultisigControl, bound to a specific deployed contract.
func NewMultisigControlCaller(address common.Address, caller bind.ContractCaller) (*MultisigControlCaller, error) {
	contract, err := bindMultisigControl(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultisigControlCaller{contract: contract}, nil
}

// NewMultisigControlTransactor creates a new write-only instance of MultisigControl, bound to a specific deployed contract.
func NewMultisigControlTransactor(address common.Address, transactor bind.ContractTransactor) (*MultisigControlTransactor, error) {
	contract, err := bindMultisigControl(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultisigControlTransactor{contract: contract}, nil
}

// NewMultisigControlFilterer creates a new log filterer instance of MultisigControl, bound to a specific deployed contract.
func NewMultisigControlFilterer(address common.Address, filterer bind.ContractFilterer) (*MultisigControlFilterer, error) {
	contract, err := bindMultisigControl(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultisigControlFilterer{contract: contract}, nil
}

// bindMultisigControl binds a generic wrapper to an already deployed contract.
func bindMultisigControl(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MultisigControlMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultisigControl *MultisigControlRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultisigControl.Contract.MultisigControlCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultisigControl *MultisigControlRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultisigControl.Contract.MultisigControlTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultisigControl *MultisigControlRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultisigControl.Contract.MultisigControlTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultisigControl *MultisigControlCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultisigControl.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultisigControl *MultisigControlTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultisigControl.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultisigControl *MultisigControlTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultisigControl.Contract.contract.Transact(opts, method, params...)
}

// GetCurrentThreshold is a free data retrieval call binding the contract method 0xdbe528df.
//
// Solidity: function get_current_threshold() view returns(uint16)
func (_MultisigControl *MultisigControlCaller) GetCurrentThreshold(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _MultisigControl.contract.Call(opts, &out, "get_current_threshold")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// GetCurrentThreshold is a free data retrieval call binding the contract method 0xdbe528df.
//
// Solidity: function get_current_threshold() view returns(uint16)
func (_MultisigControl *MultisigControlSession) GetCurrentThreshold() (uint16, error) {
	return _MultisigControl.Contract.GetCurrentThreshold(&_MultisigControl.CallOpts)
}

// GetCurrentThreshold is a free data retrieval call binding the contract method 0xdbe528df.
//
// Solidity: function get_current_threshold() view returns(uint16)
func (_MultisigControl *MultisigControlCallerSession) GetCurrentThreshold() (uint16, error) {
	return _MultisigControl.Contract.GetCurrentThreshold(&_MultisigControl.CallOpts)
}

// GetValidSignerCount is a free data retrieval call binding the contract method 0xb04e3dd1.
//
// Solidity: function get_valid_signer_count() view returns(uint8)
func (_MultisigControl *MultisigControlCaller) GetValidSignerCount(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MultisigControl.contract.Call(opts, &out, "get_valid_signer_count")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetValidSignerCount is a free data retrieval call binding the contract method 0xb04e3dd1.
//
// Solidity: function get_valid_signer_count() view returns(uint8)
func (_MultisigControl *MultisigControlSession) GetValidSignerCount() (uint8, error) {
	return _MultisigControl.Contract.GetValidSignerCount(&_MultisigControl.CallOpts)
}

// GetValidSignerCount is a free data retrieval call binding the contract method 0xb04e3dd1.
//
// Solidity: function get_valid_signer_count() view returns(uint8)
func (_MultisigControl *MultisigControlCallerSession) GetValidSignerCount() (uint8, error) {
	return _MultisigControl.Contract.GetValidSignerCount(&_MultisigControl.CallOpts)
}

// IsNonceUsed is a free data retrieval call binding the contract method 0x5b9fe26b.
//
// Solidity: function is_nonce_used(uint256 nonce) view returns(bool)
func (_MultisigControl *MultisigControlCaller) IsNonceUsed(opts *bind.CallOpts, nonce *big.Int) (bool, error) {
	var out []interface{}
	err := _MultisigControl.contract.Call(opts, &out, "is_nonce_used", nonce)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsNonceUsed is a free data retrieval call binding the contract method 0x5b9fe26b.
//
// Solidity: function is_nonce_used(uint256 nonce) view returns(bool)
func (_MultisigControl *MultisigControlSession) IsNonceUsed(nonce *big.Int) (bool, error) {
	return _MultisigControl.Contract.IsNonceUsed(&_MultisigControl.CallOpts, nonce)
}

// IsNonceUsed is a free data retrieval call binding the contract method 0x5b9fe26b.
//
// Solidity: function is_nonce_used(uint256 nonce) view returns(bool)
func (_MultisigControl *MultisigControlCallerSession) IsNonceUsed(nonce *big.Int) (bool, error) {
	return _MultisigControl.Contract.IsNonceUsed(&_MultisigControl.CallOpts, nonce)
}

// IsValidSigner is a free data retrieval call binding the contract method 0x5f061559.
//
// Solidity: function is_valid_signer(address signer_address) view returns(bool)
func (_MultisigControl *MultisigControlCaller) IsValidSigner(opts *bind.CallOpts, signer_address common.Address) (bool, error) {
	var out []interface{}
	err := _MultisigControl.contract.Call(opts, &out, "is_valid_signer", signer_address)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidSigner is a free data retrieval call binding the contract method 0x5f061559.
//
// Solidity: function is_valid_signer(address signer_address) view returns(bool)
func (_MultisigControl *MultisigControlSession) IsValidSigner(signer_address common.Address) (bool, error) {
	return _MultisigControl.Contract.IsValidSigner(&_MultisigControl.CallOpts, signer_address)
}

// IsValidSigner is a free data retrieval call binding the contract method 0x5f061559.
//
// Solidity: function is_valid_signer(address signer_address) view returns(bool)
func (_MultisigControl *MultisigControlCallerSession) IsValidSigner(signer_address common.Address) (bool, error) {
	return _MultisigControl.Contract.IsValidSigner(&_MultisigControl.CallOpts, signer_address)
}

// Signers is a free data retrieval call binding the contract method 0x736c0d5b.
//
// Solidity: function signers(address ) view returns(bool)
func (_MultisigControl *MultisigControlCaller) Signers(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _MultisigControl.contract.Call(opts, &out, "signers", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Signers is a free data retrieval call binding the contract method 0x736c0d5b.
//
// Solidity: function signers(address ) view returns(bool)
func (_MultisigControl *MultisigControlSession) Signers(arg0 common.Address) (bool, error) {
	return _MultisigControl.Contract.Signers(&_MultisigControl.CallOpts, arg0)
}

// Signers is a free data retrieval call binding the contract method 0x736c0d5b.
//
// Solidity: function signers(address ) view returns(bool)
func (_MultisigControl *MultisigControlCallerSession) Signers(arg0 common.Address) (bool, error) {
	return _MultisigControl.Contract.Signers(&_MultisigControl.CallOpts, arg0)
}

// AddSigner is a paid mutator transaction binding the contract method 0xf8e3a660.
//
// Solidity: function add_signer(address new_signer, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactor) AddSigner(opts *bind.TransactOpts, new_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.contract.Transact(opts, "add_signer", new_signer, nonce, signatures)
}

// AddSigner is a paid mutator transaction binding the contract method 0xf8e3a660.
//
// Solidity: function add_signer(address new_signer, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlSession) AddSigner(new_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.AddSigner(&_MultisigControl.TransactOpts, new_signer, nonce, signatures)
}

// AddSigner is a paid mutator transaction binding the contract method 0xf8e3a660.
//
// Solidity: function add_signer(address new_signer, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactorSession) AddSigner(new_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.AddSigner(&_MultisigControl.TransactOpts, new_signer, nonce, signatures)
}

// BurnNonce is a paid mutator transaction binding the contract method 0x5ec51639.
//
// Solidity: function burn_nonce(uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactor) BurnNonce(opts *bind.TransactOpts, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.contract.Transact(opts, "burn_nonce", nonce, signatures)
}

// BurnNonce is a paid mutator transaction binding the contract method 0x5ec51639.
//
// Solidity: function burn_nonce(uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlSession) BurnNonce(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.BurnNonce(&_MultisigControl.TransactOpts, nonce, signatures)
}

// BurnNonce is a paid mutator transaction binding the contract method 0x5ec51639.
//
// Solidity: function burn_nonce(uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactorSession) BurnNonce(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.BurnNonce(&_MultisigControl.TransactOpts, nonce, signatures)
}

// RemoveSigner is a paid mutator transaction binding the contract method 0x98c5f73e.
//
// Solidity: function remove_signer(address old_signer, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactor) RemoveSigner(opts *bind.TransactOpts, old_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.contract.Transact(opts, "remove_signer", old_signer, nonce, signatures)
}

// RemoveSigner is a paid mutator transaction binding the contract method 0x98c5f73e.
//
// Solidity: function remove_signer(address old_signer, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlSession) RemoveSigner(old_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.RemoveSigner(&_MultisigControl.TransactOpts, old_signer, nonce, signatures)
}

// RemoveSigner is a paid mutator transaction binding the contract method 0x98c5f73e.
//
// Solidity: function remove_signer(address old_signer, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactorSession) RemoveSigner(old_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.RemoveSigner(&_MultisigControl.TransactOpts, old_signer, nonce, signatures)
}

// SetThreshold is a paid mutator transaction binding the contract method 0x50ac8df8.
//
// Solidity: function set_threshold(uint16 new_threshold, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactor) SetThreshold(opts *bind.TransactOpts, new_threshold uint16, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.contract.Transact(opts, "set_threshold", new_threshold, nonce, signatures)
}

// SetThreshold is a paid mutator transaction binding the contract method 0x50ac8df8.
//
// Solidity: function set_threshold(uint16 new_threshold, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlSession) SetThreshold(new_threshold uint16, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.SetThreshold(&_MultisigControl.TransactOpts, new_threshold, nonce, signatures)
}

// SetThreshold is a paid mutator transaction binding the contract method 0x50ac8df8.
//
// Solidity: function set_threshold(uint16 new_threshold, uint256 nonce, bytes signatures) returns()
func (_MultisigControl *MultisigControlTransactorSession) SetThreshold(new_threshold uint16, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultisigControl.Contract.SetThreshold(&_MultisigControl.TransactOpts, new_threshold, nonce, signatures)
}

// VerifySignatures is a paid mutator transaction binding the contract method 0xba73659a.
//
// Solidity: function verify_signatures(bytes signatures, bytes message, uint256 nonce) returns(bool)
func (_MultisigControl *MultisigControlTransactor) VerifySignatures(opts *bind.TransactOpts, signatures []byte, message []byte, nonce *big.Int) (*types.Transaction, error) {
	return _MultisigControl.contract.Transact(opts, "verify_signatures", signatures, message, nonce)
}

// VerifySignatures is a paid mutator transaction binding the contract method 0xba73659a.
//
// Solidity: function verify_signatures(bytes signatures, bytes message, uint256 nonce) returns(bool)
func (_MultisigControl *MultisigControlSession) VerifySignatures(signatures []byte, message []byte, nonce *big.Int) (*types.Transaction, error) {
	return _MultisigControl.Contract.VerifySignatures(&_MultisigControl.TransactOpts, signatures, message, nonce)
}

// VerifySignatures is a paid mutator transaction binding the contract method 0xba73659a.
//
// Solidity: function verify_signatures(bytes signatures, bytes message, uint256 nonce) returns(bool)
func (_MultisigControl *MultisigControlTransactorSession) VerifySignatures(signatures []byte, message []byte, nonce *big.Int) (*types.Transaction, error) {
	return _MultisigControl.Contract.VerifySignatures(&_MultisigControl.TransactOpts, signatures, message, nonce)
}

// MultisigControlNonceBurntIterator is returned from FilterNonceBurnt and is used to iterate over the raw logs and unpacked data for NonceBurnt events raised by the MultisigControl contract.
type MultisigControlNonceBurntIterator struct {
	Event *MultisigControlNonceBurnt // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MultisigControlNonceBurntIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultisigControlNonceBurnt)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MultisigControlNonceBurnt)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MultisigControlNonceBurntIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultisigControlNonceBurntIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultisigControlNonceBurnt represents a NonceBurnt event raised by the MultisigControl contract.
type MultisigControlNonceBurnt struct {
	Nonce *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterNonceBurnt is a free log retrieval operation binding the contract event 0xb33a7fc220f9e1c644c0f616b48edee1956a978a7dcb37a10f16e148969e4c0b.
//
// Solidity: event NonceBurnt(uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) FilterNonceBurnt(opts *bind.FilterOpts) (*MultisigControlNonceBurntIterator, error) {

	logs, sub, err := _MultisigControl.contract.FilterLogs(opts, "NonceBurnt")
	if err != nil {
		return nil, err
	}
	return &MultisigControlNonceBurntIterator{contract: _MultisigControl.contract, event: "NonceBurnt", logs: logs, sub: sub}, nil
}

// WatchNonceBurnt is a free log subscription operation binding the contract event 0xb33a7fc220f9e1c644c0f616b48edee1956a978a7dcb37a10f16e148969e4c0b.
//
// Solidity: event NonceBurnt(uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) WatchNonceBurnt(opts *bind.WatchOpts, sink chan<- *MultisigControlNonceBurnt) (event.Subscription, error) {

	logs, sub, err := _MultisigControl.contract.WatchLogs(opts, "NonceBurnt")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultisigControlNonceBurnt)
				if err := _MultisigControl.contract.UnpackLog(event, "NonceBurnt", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseNonceBurnt is a log parse operation binding the contract event 0xb33a7fc220f9e1c644c0f616b48edee1956a978a7dcb37a10f16e148969e4c0b.
//
// Solidity: event NonceBurnt(uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) ParseNonceBurnt(log types.Log) (*MultisigControlNonceBurnt, error) {
	event := new(MultisigControlNonceBurnt)
	if err := _MultisigControl.contract.UnpackLog(event, "NonceBurnt", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MultisigControlSignerAddedIterator is returned from FilterSignerAdded and is used to iterate over the raw logs and unpacked data for SignerAdded events raised by the MultisigControl contract.
type MultisigControlSignerAddedIterator struct {
	Event *MultisigControlSignerAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MultisigControlSignerAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultisigControlSignerAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MultisigControlSignerAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MultisigControlSignerAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultisigControlSignerAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultisigControlSignerAdded represents a SignerAdded event raised by the MultisigControl contract.
type MultisigControlSignerAdded struct {
	NewSigner common.Address
	Nonce     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSignerAdded is a free log retrieval operation binding the contract event 0x50999ebf9b59bf3157a58816611976f2d723378ad51457d7b0413209e0cdee59.
//
// Solidity: event SignerAdded(address new_signer, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) FilterSignerAdded(opts *bind.FilterOpts) (*MultisigControlSignerAddedIterator, error) {

	logs, sub, err := _MultisigControl.contract.FilterLogs(opts, "SignerAdded")
	if err != nil {
		return nil, err
	}
	return &MultisigControlSignerAddedIterator{contract: _MultisigControl.contract, event: "SignerAdded", logs: logs, sub: sub}, nil
}

// WatchSignerAdded is a free log subscription operation binding the contract event 0x50999ebf9b59bf3157a58816611976f2d723378ad51457d7b0413209e0cdee59.
//
// Solidity: event SignerAdded(address new_signer, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) WatchSignerAdded(opts *bind.WatchOpts, sink chan<- *MultisigControlSignerAdded) (event.Subscription, error) {

	logs, sub, err := _MultisigControl.contract.WatchLogs(opts, "SignerAdded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultisigControlSignerAdded)
				if err := _MultisigControl.contract.UnpackLog(event, "SignerAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSignerAdded is a log parse operation binding the contract event 0x50999ebf9b59bf3157a58816611976f2d723378ad51457d7b0413209e0cdee59.
//
// Solidity: event SignerAdded(address new_signer, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) ParseSignerAdded(log types.Log) (*MultisigControlSignerAdded, error) {
	event := new(MultisigControlSignerAdded)
	if err := _MultisigControl.contract.UnpackLog(event, "SignerAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MultisigControlSignerRemovedIterator is returned from FilterSignerRemoved and is used to iterate over the raw logs and unpacked data for SignerRemoved events raised by the MultisigControl contract.
type MultisigControlSignerRemovedIterator struct {
	Event *MultisigControlSignerRemoved // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MultisigControlSignerRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultisigControlSignerRemoved)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MultisigControlSignerRemoved)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MultisigControlSignerRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultisigControlSignerRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultisigControlSignerRemoved represents a SignerRemoved event raised by the MultisigControl contract.
type MultisigControlSignerRemoved struct {
	OldSigner common.Address
	Nonce     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSignerRemoved is a free log retrieval operation binding the contract event 0x99c1d2c0ed8107e4db2e5dbfb10a2549cd2a63cbe39cf99d2adffbcd03954418.
//
// Solidity: event SignerRemoved(address old_signer, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) FilterSignerRemoved(opts *bind.FilterOpts) (*MultisigControlSignerRemovedIterator, error) {

	logs, sub, err := _MultisigControl.contract.FilterLogs(opts, "SignerRemoved")
	if err != nil {
		return nil, err
	}
	return &MultisigControlSignerRemovedIterator{contract: _MultisigControl.contract, event: "SignerRemoved", logs: logs, sub: sub}, nil
}

// WatchSignerRemoved is a free log subscription operation binding the contract event 0x99c1d2c0ed8107e4db2e5dbfb10a2549cd2a63cbe39cf99d2adffbcd03954418.
//
// Solidity: event SignerRemoved(address old_signer, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) WatchSignerRemoved(opts *bind.WatchOpts, sink chan<- *MultisigControlSignerRemoved) (event.Subscription, error) {

	logs, sub, err := _MultisigControl.contract.WatchLogs(opts, "SignerRemoved")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultisigControlSignerRemoved)
				if err := _MultisigControl.contract.UnpackLog(event, "SignerRemoved", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSignerRemoved is a log parse operation binding the contract event 0x99c1d2c0ed8107e4db2e5dbfb10a2549cd2a63cbe39cf99d2adffbcd03954418.
//
// Solidity: event SignerRemoved(address old_signer, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) ParseSignerRemoved(log types.Log) (*MultisigControlSignerRemoved, error) {
	event := new(MultisigControlSignerRemoved)
	if err := _MultisigControl.contract.UnpackLog(event, "SignerRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MultisigControlThresholdSetIterator is returned from FilterThresholdSet and is used to iterate over the raw logs and unpacked data for ThresholdSet events raised by the MultisigControl contract.
type MultisigControlThresholdSetIterator struct {
	Event *MultisigControlThresholdSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *MultisigControlThresholdSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultisigControlThresholdSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(MultisigControlThresholdSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *MultisigControlThresholdSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultisigControlThresholdSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultisigControlThresholdSet represents a ThresholdSet event raised by the MultisigControl contract.
type MultisigControlThresholdSet struct {
	NewThreshold uint16
	Nonce        *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterThresholdSet is a free log retrieval operation binding the contract event 0xf6d24c23627520a3b70e5dc66aa1249844b4bb407c2c153d9000a2b14a1e3c11.
//
// Solidity: event ThresholdSet(uint16 new_threshold, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) FilterThresholdSet(opts *bind.FilterOpts) (*MultisigControlThresholdSetIterator, error) {

	logs, sub, err := _MultisigControl.contract.FilterLogs(opts, "ThresholdSet")
	if err != nil {
		return nil, err
	}
	return &MultisigControlThresholdSetIterator{contract: _MultisigControl.contract, event: "ThresholdSet", logs: logs, sub: sub}, nil
}

// WatchThresholdSet is a free log subscription operation binding the contract event 0xf6d24c23627520a3b70e5dc66aa1249844b4bb407c2c153d9000a2b14a1e3c11.
//
// Solidity: event ThresholdSet(uint16 new_threshold, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) WatchThresholdSet(opts *bind.WatchOpts, sink chan<- *MultisigControlThresholdSet) (event.Subscription, error) {

	logs, sub, err := _MultisigControl.contract.WatchLogs(opts, "ThresholdSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultisigControlThresholdSet)
				if err := _MultisigControl.contract.UnpackLog(event, "ThresholdSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseThresholdSet is a log parse operation binding the contract event 0xf6d24c23627520a3b70e5dc66aa1249844b4bb407c2c153d9000a2b14a1e3c11.
//
// Solidity: event ThresholdSet(uint16 new_threshold, uint256 nonce)
func (_MultisigControl *MultisigControlFilterer) ParseThresholdSet(log types.Log) (*MultisigControlThresholdSet, error) {
	event := new(MultisigControlThresholdSet)
	if err := _MultisigControl.contract.UnpackLog(event, "ThresholdSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
