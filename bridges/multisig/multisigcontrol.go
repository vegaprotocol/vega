// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package multisig

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
)

// MultiSigControlMetaData contains all meta data concerning the MultiSigControl contract.
var MultiSigControlMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"new_signer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"SignerAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"old_signer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"SignerRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint16\",\"name\":\"new_threshold\",\"type\":\"uint16\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"ThresholdSet\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"new_signer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"add_signer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_current_threshold\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_valid_signer_count\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"is_nonce_used\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"signer_address\",\"type\":\"address\"}],\"name\":\"is_valid_signer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"old_signer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"remove_signer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint16\",\"name\":\"new_threshold\",\"type\":\"uint16\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"set_threshold\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"verify_signatures\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// MultiSigControlABI is the input ABI used to generate the binding from.
// Deprecated: Use MultiSigControlMetaData.ABI instead.
var MultiSigControlABI = MultiSigControlMetaData.ABI

// MultiSigControl is an auto generated Go binding around an Ethereum contract.
type MultiSigControl struct {
	MultiSigControlCaller     // Read-only binding to the contract
	MultiSigControlTransactor // Write-only binding to the contract
	MultiSigControlFilterer   // Log filterer for contract events
}

// MultiSigControlCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultiSigControlCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSigControlTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiSigControlTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSigControlFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiSigControlFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSigControlSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiSigControlSession struct {
	Contract     *MultiSigControl  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MultiSigControlCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiSigControlCallerSession struct {
	Contract *MultiSigControlCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// MultiSigControlTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiSigControlTransactorSession struct {
	Contract     *MultiSigControlTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// MultiSigControlRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultiSigControlRaw struct {
	Contract *MultiSigControl // Generic contract binding to access the raw methods on
}

// MultiSigControlCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiSigControlCallerRaw struct {
	Contract *MultiSigControlCaller // Generic read-only contract binding to access the raw methods on
}

// MultiSigControlTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiSigControlTransactorRaw struct {
	Contract *MultiSigControlTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiSigControl creates a new instance of MultiSigControl, bound to a specific deployed contract.
func NewMultiSigControl(address common.Address, backend bind.ContractBackend) (*MultiSigControl, error) {
	contract, err := bindMultiSigControl(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiSigControl{MultiSigControlCaller: MultiSigControlCaller{contract: contract}, MultiSigControlTransactor: MultiSigControlTransactor{contract: contract}, MultiSigControlFilterer: MultiSigControlFilterer{contract: contract}}, nil
}

// NewMultiSigControlCaller creates a new read-only instance of MultiSigControl, bound to a specific deployed contract.
func NewMultiSigControlCaller(address common.Address, caller bind.ContractCaller) (*MultiSigControlCaller, error) {
	contract, err := bindMultiSigControl(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSigControlCaller{contract: contract}, nil
}

// NewMultiSigControlTransactor creates a new write-only instance of MultiSigControl, bound to a specific deployed contract.
func NewMultiSigControlTransactor(address common.Address, transactor bind.ContractTransactor) (*MultiSigControlTransactor, error) {
	contract, err := bindMultiSigControl(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSigControlTransactor{contract: contract}, nil
}

// NewMultiSigControlFilterer creates a new log filterer instance of MultiSigControl, bound to a specific deployed contract.
func NewMultiSigControlFilterer(address common.Address, filterer bind.ContractFilterer) (*MultiSigControlFilterer, error) {
	contract, err := bindMultiSigControl(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiSigControlFilterer{contract: contract}, nil
}

// bindMultiSigControl binds a generic wrapper to an already deployed contract.
func bindMultiSigControl(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiSigControlABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSigControl *MultiSigControlRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSigControl.Contract.MultiSigControlCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSigControl *MultiSigControlRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSigControl.Contract.MultiSigControlTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSigControl *MultiSigControlRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSigControl.Contract.MultiSigControlTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSigControl *MultiSigControlCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSigControl.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSigControl *MultiSigControlTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSigControl.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSigControl *MultiSigControlTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSigControl.Contract.contract.Transact(opts, method, params...)
}

// GetCurrentThreshold is a free data retrieval call binding the contract method 0xdbe528df.
//
// Solidity: function get_current_threshold() view returns(uint16)
func (_MultiSigControl *MultiSigControlCaller) GetCurrentThreshold(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _MultiSigControl.contract.Call(opts, &out, "get_current_threshold")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// GetCurrentThreshold is a free data retrieval call binding the contract method 0xdbe528df.
//
// Solidity: function get_current_threshold() view returns(uint16)
func (_MultiSigControl *MultiSigControlSession) GetCurrentThreshold() (uint16, error) {
	return _MultiSigControl.Contract.GetCurrentThreshold(&_MultiSigControl.CallOpts)
}

// GetCurrentThreshold is a free data retrieval call binding the contract method 0xdbe528df.
//
// Solidity: function get_current_threshold() view returns(uint16)
func (_MultiSigControl *MultiSigControlCallerSession) GetCurrentThreshold() (uint16, error) {
	return _MultiSigControl.Contract.GetCurrentThreshold(&_MultiSigControl.CallOpts)
}

// GetValidSignerCount is a free data retrieval call binding the contract method 0xb04e3dd1.
//
// Solidity: function get_valid_signer_count() view returns(uint8)
func (_MultiSigControl *MultiSigControlCaller) GetValidSignerCount(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _MultiSigControl.contract.Call(opts, &out, "get_valid_signer_count")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetValidSignerCount is a free data retrieval call binding the contract method 0xb04e3dd1.
//
// Solidity: function get_valid_signer_count() view returns(uint8)
func (_MultiSigControl *MultiSigControlSession) GetValidSignerCount() (uint8, error) {
	return _MultiSigControl.Contract.GetValidSignerCount(&_MultiSigControl.CallOpts)
}

// GetValidSignerCount is a free data retrieval call binding the contract method 0xb04e3dd1.
//
// Solidity: function get_valid_signer_count() view returns(uint8)
func (_MultiSigControl *MultiSigControlCallerSession) GetValidSignerCount() (uint8, error) {
	return _MultiSigControl.Contract.GetValidSignerCount(&_MultiSigControl.CallOpts)
}

// IsNonceUsed is a free data retrieval call binding the contract method 0x5b9fe26b.
//
// Solidity: function is_nonce_used(uint256 nonce) view returns(bool)
func (_MultiSigControl *MultiSigControlCaller) IsNonceUsed(opts *bind.CallOpts, nonce *big.Int) (bool, error) {
	var out []interface{}
	err := _MultiSigControl.contract.Call(opts, &out, "is_nonce_used", nonce)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsNonceUsed is a free data retrieval call binding the contract method 0x5b9fe26b.
//
// Solidity: function is_nonce_used(uint256 nonce) view returns(bool)
func (_MultiSigControl *MultiSigControlSession) IsNonceUsed(nonce *big.Int) (bool, error) {
	return _MultiSigControl.Contract.IsNonceUsed(&_MultiSigControl.CallOpts, nonce)
}

// IsNonceUsed is a free data retrieval call binding the contract method 0x5b9fe26b.
//
// Solidity: function is_nonce_used(uint256 nonce) view returns(bool)
func (_MultiSigControl *MultiSigControlCallerSession) IsNonceUsed(nonce *big.Int) (bool, error) {
	return _MultiSigControl.Contract.IsNonceUsed(&_MultiSigControl.CallOpts, nonce)
}

// IsValidSigner is a free data retrieval call binding the contract method 0x5f061559.
//
// Solidity: function is_valid_signer(address signer_address) view returns(bool)
func (_MultiSigControl *MultiSigControlCaller) IsValidSigner(opts *bind.CallOpts, signer_address common.Address) (bool, error) {
	var out []interface{}
	err := _MultiSigControl.contract.Call(opts, &out, "is_valid_signer", signer_address)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidSigner is a free data retrieval call binding the contract method 0x5f061559.
//
// Solidity: function is_valid_signer(address signer_address) view returns(bool)
func (_MultiSigControl *MultiSigControlSession) IsValidSigner(signer_address common.Address) (bool, error) {
	return _MultiSigControl.Contract.IsValidSigner(&_MultiSigControl.CallOpts, signer_address)
}

// IsValidSigner is a free data retrieval call binding the contract method 0x5f061559.
//
// Solidity: function is_valid_signer(address signer_address) view returns(bool)
func (_MultiSigControl *MultiSigControlCallerSession) IsValidSigner(signer_address common.Address) (bool, error) {
	return _MultiSigControl.Contract.IsValidSigner(&_MultiSigControl.CallOpts, signer_address)
}

// AddSigner is a paid mutator transaction binding the contract method 0xf8e3a660.
//
// Solidity: function add_signer(address new_signer, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlTransactor) AddSigner(opts *bind.TransactOpts, new_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.contract.Transact(opts, "add_signer", new_signer, nonce, signatures)
}

// AddSigner is a paid mutator transaction binding the contract method 0xf8e3a660.
//
// Solidity: function add_signer(address new_signer, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlSession) AddSigner(new_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.Contract.AddSigner(&_MultiSigControl.TransactOpts, new_signer, nonce, signatures)
}

// AddSigner is a paid mutator transaction binding the contract method 0xf8e3a660.
//
// Solidity: function add_signer(address new_signer, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlTransactorSession) AddSigner(new_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.Contract.AddSigner(&_MultiSigControl.TransactOpts, new_signer, nonce, signatures)
}

// RemoveSigner is a paid mutator transaction binding the contract method 0x98c5f73e.
//
// Solidity: function remove_signer(address old_signer, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlTransactor) RemoveSigner(opts *bind.TransactOpts, old_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.contract.Transact(opts, "remove_signer", old_signer, nonce, signatures)
}

// RemoveSigner is a paid mutator transaction binding the contract method 0x98c5f73e.
//
// Solidity: function remove_signer(address old_signer, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlSession) RemoveSigner(old_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.Contract.RemoveSigner(&_MultiSigControl.TransactOpts, old_signer, nonce, signatures)
}

// RemoveSigner is a paid mutator transaction binding the contract method 0x98c5f73e.
//
// Solidity: function remove_signer(address old_signer, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlTransactorSession) RemoveSigner(old_signer common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.Contract.RemoveSigner(&_MultiSigControl.TransactOpts, old_signer, nonce, signatures)
}

// SetThreshold is a paid mutator transaction binding the contract method 0x50ac8df8.
//
// Solidity: function set_threshold(uint16 new_threshold, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlTransactor) SetThreshold(opts *bind.TransactOpts, new_threshold uint16, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.contract.Transact(opts, "set_threshold", new_threshold, nonce, signatures)
}

// SetThreshold is a paid mutator transaction binding the contract method 0x50ac8df8.
//
// Solidity: function set_threshold(uint16 new_threshold, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlSession) SetThreshold(new_threshold uint16, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.Contract.SetThreshold(&_MultiSigControl.TransactOpts, new_threshold, nonce, signatures)
}

// SetThreshold is a paid mutator transaction binding the contract method 0x50ac8df8.
//
// Solidity: function set_threshold(uint16 new_threshold, uint256 nonce, bytes signatures) returns()
func (_MultiSigControl *MultiSigControlTransactorSession) SetThreshold(new_threshold uint16, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _MultiSigControl.Contract.SetThreshold(&_MultiSigControl.TransactOpts, new_threshold, nonce, signatures)
}

// VerifySignatures is a paid mutator transaction binding the contract method 0xba73659a.
//
// Solidity: function verify_signatures(bytes signatures, bytes message, uint256 nonce) returns(bool)
func (_MultiSigControl *MultiSigControlTransactor) VerifySignatures(opts *bind.TransactOpts, signatures []byte, message []byte, nonce *big.Int) (*types.Transaction, error) {
	return _MultiSigControl.contract.Transact(opts, "verify_signatures", signatures, message, nonce)
}

// VerifySignatures is a paid mutator transaction binding the contract method 0xba73659a.
//
// Solidity: function verify_signatures(bytes signatures, bytes message, uint256 nonce) returns(bool)
func (_MultiSigControl *MultiSigControlSession) VerifySignatures(signatures []byte, message []byte, nonce *big.Int) (*types.Transaction, error) {
	return _MultiSigControl.Contract.VerifySignatures(&_MultiSigControl.TransactOpts, signatures, message, nonce)
}

// VerifySignatures is a paid mutator transaction binding the contract method 0xba73659a.
//
// Solidity: function verify_signatures(bytes signatures, bytes message, uint256 nonce) returns(bool)
func (_MultiSigControl *MultiSigControlTransactorSession) VerifySignatures(signatures []byte, message []byte, nonce *big.Int) (*types.Transaction, error) {
	return _MultiSigControl.Contract.VerifySignatures(&_MultiSigControl.TransactOpts, signatures, message, nonce)
}

// MultiSigControlSignerAddedIterator is returned from FilterSignerAdded and is used to iterate over the raw logs and unpacked data for SignerAdded events raised by the MultiSigControl contract.
type MultiSigControlSignerAddedIterator struct {
	Event *MultiSigControlSignerAdded // Event containing the contract specifics and raw log

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
func (it *MultiSigControlSignerAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiSigControlSignerAdded)
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
		it.Event = new(MultiSigControlSignerAdded)
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
func (it *MultiSigControlSignerAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiSigControlSignerAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiSigControlSignerAdded represents a SignerAdded event raised by the MultiSigControl contract.
type MultiSigControlSignerAdded struct {
	NewSigner common.Address
	Nonce     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSignerAdded is a free log retrieval operation binding the contract event 0x50999ebf9b59bf3157a58816611976f2d723378ad51457d7b0413209e0cdee59.
//
// Solidity: event SignerAdded(address new_signer, uint256 nonce)
func (_MultiSigControl *MultiSigControlFilterer) FilterSignerAdded(opts *bind.FilterOpts) (*MultiSigControlSignerAddedIterator, error) {

	logs, sub, err := _MultiSigControl.contract.FilterLogs(opts, "SignerAdded")
	if err != nil {
		return nil, err
	}
	return &MultiSigControlSignerAddedIterator{contract: _MultiSigControl.contract, event: "SignerAdded", logs: logs, sub: sub}, nil
}

// WatchSignerAdded is a free log subscription operation binding the contract event 0x50999ebf9b59bf3157a58816611976f2d723378ad51457d7b0413209e0cdee59.
//
// Solidity: event SignerAdded(address new_signer, uint256 nonce)
func (_MultiSigControl *MultiSigControlFilterer) WatchSignerAdded(opts *bind.WatchOpts, sink chan<- *MultiSigControlSignerAdded) (event.Subscription, error) {

	logs, sub, err := _MultiSigControl.contract.WatchLogs(opts, "SignerAdded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiSigControlSignerAdded)
				if err := _MultiSigControl.contract.UnpackLog(event, "SignerAdded", log); err != nil {
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
func (_MultiSigControl *MultiSigControlFilterer) ParseSignerAdded(log types.Log) (*MultiSigControlSignerAdded, error) {
	event := new(MultiSigControlSignerAdded)
	if err := _MultiSigControl.contract.UnpackLog(event, "SignerAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MultiSigControlSignerRemovedIterator is returned from FilterSignerRemoved and is used to iterate over the raw logs and unpacked data for SignerRemoved events raised by the MultiSigControl contract.
type MultiSigControlSignerRemovedIterator struct {
	Event *MultiSigControlSignerRemoved // Event containing the contract specifics and raw log

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
func (it *MultiSigControlSignerRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiSigControlSignerRemoved)
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
		it.Event = new(MultiSigControlSignerRemoved)
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
func (it *MultiSigControlSignerRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiSigControlSignerRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiSigControlSignerRemoved represents a SignerRemoved event raised by the MultiSigControl contract.
type MultiSigControlSignerRemoved struct {
	OldSigner common.Address
	Nonce     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterSignerRemoved is a free log retrieval operation binding the contract event 0x99c1d2c0ed8107e4db2e5dbfb10a2549cd2a63cbe39cf99d2adffbcd03954418.
//
// Solidity: event SignerRemoved(address old_signer, uint256 nonce)
func (_MultiSigControl *MultiSigControlFilterer) FilterSignerRemoved(opts *bind.FilterOpts) (*MultiSigControlSignerRemovedIterator, error) {

	logs, sub, err := _MultiSigControl.contract.FilterLogs(opts, "SignerRemoved")
	if err != nil {
		return nil, err
	}
	return &MultiSigControlSignerRemovedIterator{contract: _MultiSigControl.contract, event: "SignerRemoved", logs: logs, sub: sub}, nil
}

// WatchSignerRemoved is a free log subscription operation binding the contract event 0x99c1d2c0ed8107e4db2e5dbfb10a2549cd2a63cbe39cf99d2adffbcd03954418.
//
// Solidity: event SignerRemoved(address old_signer, uint256 nonce)
func (_MultiSigControl *MultiSigControlFilterer) WatchSignerRemoved(opts *bind.WatchOpts, sink chan<- *MultiSigControlSignerRemoved) (event.Subscription, error) {

	logs, sub, err := _MultiSigControl.contract.WatchLogs(opts, "SignerRemoved")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiSigControlSignerRemoved)
				if err := _MultiSigControl.contract.UnpackLog(event, "SignerRemoved", log); err != nil {
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
func (_MultiSigControl *MultiSigControlFilterer) ParseSignerRemoved(log types.Log) (*MultiSigControlSignerRemoved, error) {
	event := new(MultiSigControlSignerRemoved)
	if err := _MultiSigControl.contract.UnpackLog(event, "SignerRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MultiSigControlThresholdSetIterator is returned from FilterThresholdSet and is used to iterate over the raw logs and unpacked data for ThresholdSet events raised by the MultiSigControl contract.
type MultiSigControlThresholdSetIterator struct {
	Event *MultiSigControlThresholdSet // Event containing the contract specifics and raw log

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
func (it *MultiSigControlThresholdSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MultiSigControlThresholdSet)
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
		it.Event = new(MultiSigControlThresholdSet)
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
func (it *MultiSigControlThresholdSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MultiSigControlThresholdSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MultiSigControlThresholdSet represents a ThresholdSet event raised by the MultiSigControl contract.
type MultiSigControlThresholdSet struct {
	NewThreshold uint16
	Nonce        *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterThresholdSet is a free log retrieval operation binding the contract event 0xf6d24c23627520a3b70e5dc66aa1249844b4bb407c2c153d9000a2b14a1e3c11.
//
// Solidity: event ThresholdSet(uint16 new_threshold, uint256 nonce)
func (_MultiSigControl *MultiSigControlFilterer) FilterThresholdSet(opts *bind.FilterOpts) (*MultiSigControlThresholdSetIterator, error) {

	logs, sub, err := _MultiSigControl.contract.FilterLogs(opts, "ThresholdSet")
	if err != nil {
		return nil, err
	}
	return &MultiSigControlThresholdSetIterator{contract: _MultiSigControl.contract, event: "ThresholdSet", logs: logs, sub: sub}, nil
}

// WatchThresholdSet is a free log subscription operation binding the contract event 0xf6d24c23627520a3b70e5dc66aa1249844b4bb407c2c153d9000a2b14a1e3c11.
//
// Solidity: event ThresholdSet(uint16 new_threshold, uint256 nonce)
func (_MultiSigControl *MultiSigControlFilterer) WatchThresholdSet(opts *bind.WatchOpts, sink chan<- *MultiSigControlThresholdSet) (event.Subscription, error) {

	logs, sub, err := _MultiSigControl.contract.WatchLogs(opts, "ThresholdSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MultiSigControlThresholdSet)
				if err := _MultiSigControl.contract.UnpackLog(event, "ThresholdSet", log); err != nil {
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
func (_MultiSigControl *MultiSigControlFilterer) ParseThresholdSet(log types.Log) (*MultiSigControlThresholdSet, error) {
	event := new(MultiSigControlThresholdSet)
	if err := _MultiSigControl.contract.UnpackLog(event, "ThresholdSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
