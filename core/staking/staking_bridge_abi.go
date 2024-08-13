// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package staking

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

// StakingMetaData contains all meta data concerning the Staking contract.
var StakingMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"StakeDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"StakeRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"Stake_Transferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"staking_token\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"stake_balance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"total_staked\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// StakingABI is the input ABI used to generate the binding from.
// Deprecated: Use StakingMetaData.ABI instead.
var StakingABI = StakingMetaData.ABI

// Staking is an auto generated Go binding around an Ethereum contract.
type Staking struct {
	StakingCaller     // Read-only binding to the contract
	StakingTransactor // Write-only binding to the contract
	StakingFilterer   // Log filterer for contract events
}

// StakingCaller is an auto generated read-only Go binding around an Ethereum contract.
type StakingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StakingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StakingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StakingSession struct {
	Contract     *Staking          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StakingCallerSession struct {
	Contract *StakingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// StakingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StakingTransactorSession struct {
	Contract     *StakingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// StakingRaw is an auto generated low-level Go binding around an Ethereum contract.
type StakingRaw struct {
	Contract *Staking // Generic contract binding to access the raw methods on
}

// StakingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StakingCallerRaw struct {
	Contract *StakingCaller // Generic read-only contract binding to access the raw methods on
}

// StakingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StakingTransactorRaw struct {
	Contract *StakingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStaking creates a new instance of Staking, bound to a specific deployed contract.
func NewStaking(address common.Address, backend bind.ContractBackend) (*Staking, error) {
	contract, err := bindStaking(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Staking{StakingCaller: StakingCaller{contract: contract}, StakingTransactor: StakingTransactor{contract: contract}, StakingFilterer: StakingFilterer{contract: contract}}, nil
}

// NewStakingCaller creates a new read-only instance of Staking, bound to a specific deployed contract.
func NewStakingCaller(address common.Address, caller bind.ContractCaller) (*StakingCaller, error) {
	contract, err := bindStaking(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StakingCaller{contract: contract}, nil
}

// NewStakingTransactor creates a new write-only instance of Staking, bound to a specific deployed contract.
func NewStakingTransactor(address common.Address, transactor bind.ContractTransactor) (*StakingTransactor, error) {
	contract, err := bindStaking(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StakingTransactor{contract: contract}, nil
}

// NewStakingFilterer creates a new log filterer instance of Staking, bound to a specific deployed contract.
func NewStakingFilterer(address common.Address, filterer bind.ContractFilterer) (*StakingFilterer, error) {
	contract, err := bindStaking(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StakingFilterer{contract: contract}, nil
}

// bindStaking binds a generic wrapper to an already deployed contract.
func bindStaking(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(StakingABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Staking *StakingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Staking.Contract.StakingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Staking *StakingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staking.Contract.StakingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Staking *StakingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Staking.Contract.StakingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Staking *StakingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Staking.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Staking *StakingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staking.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Staking *StakingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Staking.Contract.contract.Transact(opts, method, params...)
}

// StakeBalance is a free data retrieval call binding the contract method 0x274abf34.
//
// Solidity: function stake_balance(address target, bytes32 vega_public_key) view returns(uint256)
func (_Staking *StakingCaller) StakeBalance(opts *bind.CallOpts, target common.Address, vega_public_key [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staking.contract.Call(opts, &out, "stake_balance", target, vega_public_key)
	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err
}

// StakeBalance is a free data retrieval call binding the contract method 0x274abf34.
//
// Solidity: function stake_balance(address target, bytes32 vega_public_key) view returns(uint256)
func (_Staking *StakingSession) StakeBalance(target common.Address, vega_public_key [32]byte) (*big.Int, error) {
	return _Staking.Contract.StakeBalance(&_Staking.CallOpts, target, vega_public_key)
}

// StakeBalance is a free data retrieval call binding the contract method 0x274abf34.
//
// Solidity: function stake_balance(address target, bytes32 vega_public_key) view returns(uint256)
func (_Staking *StakingCallerSession) StakeBalance(target common.Address, vega_public_key [32]byte) (*big.Int, error) {
	return _Staking.Contract.StakeBalance(&_Staking.CallOpts, target, vega_public_key)
}

// StakingToken is a free data retrieval call binding the contract method 0x2dc7d74c.
//
// Solidity: function staking_token() view returns(address)
func (_Staking *StakingCaller) StakingToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Staking.contract.Call(opts, &out, "staking_token")
	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err
}

// StakingToken is a free data retrieval call binding the contract method 0x2dc7d74c.
//
// Solidity: function staking_token() view returns(address)
func (_Staking *StakingSession) StakingToken() (common.Address, error) {
	return _Staking.Contract.StakingToken(&_Staking.CallOpts)
}

// StakingToken is a free data retrieval call binding the contract method 0x2dc7d74c.
//
// Solidity: function staking_token() view returns(address)
func (_Staking *StakingCallerSession) StakingToken() (common.Address, error) {
	return _Staking.Contract.StakingToken(&_Staking.CallOpts)
}

// TotalStaked is a free data retrieval call binding the contract method 0xaf7568dd.
//
// Solidity: function total_staked() view returns(uint256)
func (_Staking *StakingCaller) TotalStaked(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staking.contract.Call(opts, &out, "total_staked")
	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err
}

// TotalStaked is a free data retrieval call binding the contract method 0xaf7568dd.
//
// Solidity: function total_staked() view returns(uint256)
func (_Staking *StakingSession) TotalStaked() (*big.Int, error) {
	return _Staking.Contract.TotalStaked(&_Staking.CallOpts)
}

// TotalStaked is a free data retrieval call binding the contract method 0xaf7568dd.
//
// Solidity: function total_staked() view returns(uint256)
func (_Staking *StakingCallerSession) TotalStaked() (*big.Int, error) {
	return _Staking.Contract.TotalStaked(&_Staking.CallOpts)
}

// StakingStakeDepositedIterator is returned from FilterStakeDeposited and is used to iterate over the raw logs and unpacked data for StakeDeposited events raised by the Staking contract.
type StakingStakeDepositedIterator struct {
	Event *StakingStakeDeposited // Event containing the contract specifics and raw log

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
func (it *StakingStakeDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakingStakeDeposited)
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
		it.Event = new(StakingStakeDeposited)
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
func (it *StakingStakeDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakingStakeDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakingStakeDeposited represents a StakeDeposited event raised by the Staking contract.
type StakingStakeDeposited struct {
	User          common.Address
	Amount        *big.Int
	VegaPublicKey [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterStakeDeposited is a free log retrieval operation binding the contract event 0x9e3e33edf5dcded4adabc51b1266225d00fa41516bfcad69513fa4eca69519da.
//
// Solidity: event StakeDeposited(address indexed user, uint256 amount, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) FilterStakeDeposited(opts *bind.FilterOpts, user []common.Address, vega_public_key [][32]byte) (*StakingStakeDepositedIterator, error) {
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	var vega_public_keyRule []interface{}
	for _, vega_public_keyItem := range vega_public_key {
		vega_public_keyRule = append(vega_public_keyRule, vega_public_keyItem)
	}

	logs, sub, err := _Staking.contract.FilterLogs(opts, "StakeDeposited", userRule, vega_public_keyRule)
	if err != nil {
		return nil, err
	}
	return &StakingStakeDepositedIterator{contract: _Staking.contract, event: "StakeDeposited", logs: logs, sub: sub}, nil
}

// WatchStakeDeposited is a free log subscription operation binding the contract event 0x9e3e33edf5dcded4adabc51b1266225d00fa41516bfcad69513fa4eca69519da.
//
// Solidity: event StakeDeposited(address indexed user, uint256 amount, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) WatchStakeDeposited(opts *bind.WatchOpts, sink chan<- *StakingStakeDeposited, user []common.Address, vega_public_key [][32]byte) (event.Subscription, error) {
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	var vega_public_keyRule []interface{}
	for _, vega_public_keyItem := range vega_public_key {
		vega_public_keyRule = append(vega_public_keyRule, vega_public_keyItem)
	}

	logs, sub, err := _Staking.contract.WatchLogs(opts, "StakeDeposited", userRule, vega_public_keyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakingStakeDeposited)
				if err := _Staking.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
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

// ParseStakeDeposited is a log parse operation binding the contract event 0x9e3e33edf5dcded4adabc51b1266225d00fa41516bfcad69513fa4eca69519da.
//
// Solidity: event StakeDeposited(address indexed user, uint256 amount, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) ParseStakeDeposited(log types.Log) (*StakingStakeDeposited, error) {
	event := new(StakingStakeDeposited)
	if err := _Staking.contract.UnpackLog(event, "StakeDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakingStakeRemovedIterator is returned from FilterStakeRemoved and is used to iterate over the raw logs and unpacked data for StakeRemoved events raised by the Staking contract.
type StakingStakeRemovedIterator struct {
	Event *StakingStakeRemoved // Event containing the contract specifics and raw log

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
func (it *StakingStakeRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakingStakeRemoved)
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
		it.Event = new(StakingStakeRemoved)
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
func (it *StakingStakeRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakingStakeRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakingStakeRemoved represents a StakeRemoved event raised by the Staking contract.
type StakingStakeRemoved struct {
	User          common.Address
	Amount        *big.Int
	VegaPublicKey [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterStakeRemoved is a free log retrieval operation binding the contract event 0xa131d16963736e4c641f27a7f82f2e350b5971e555ae06ae906892bbba0a0939.
//
// Solidity: event StakeRemoved(address indexed user, uint256 amount, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) FilterStakeRemoved(opts *bind.FilterOpts, user []common.Address, vega_public_key [][32]byte) (*StakingStakeRemovedIterator, error) {
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	var vega_public_keyRule []interface{}
	for _, vega_public_keyItem := range vega_public_key {
		vega_public_keyRule = append(vega_public_keyRule, vega_public_keyItem)
	}

	logs, sub, err := _Staking.contract.FilterLogs(opts, "StakeRemoved", userRule, vega_public_keyRule)
	if err != nil {
		return nil, err
	}
	return &StakingStakeRemovedIterator{contract: _Staking.contract, event: "StakeRemoved", logs: logs, sub: sub}, nil
}

// WatchStakeRemoved is a free log subscription operation binding the contract event 0xa131d16963736e4c641f27a7f82f2e350b5971e555ae06ae906892bbba0a0939.
//
// Solidity: event StakeRemoved(address indexed user, uint256 amount, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) WatchStakeRemoved(opts *bind.WatchOpts, sink chan<- *StakingStakeRemoved, user []common.Address, vega_public_key [][32]byte) (event.Subscription, error) {
	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	var vega_public_keyRule []interface{}
	for _, vega_public_keyItem := range vega_public_key {
		vega_public_keyRule = append(vega_public_keyRule, vega_public_keyItem)
	}

	logs, sub, err := _Staking.contract.WatchLogs(opts, "StakeRemoved", userRule, vega_public_keyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakingStakeRemoved)
				if err := _Staking.contract.UnpackLog(event, "StakeRemoved", log); err != nil {
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

// ParseStakeRemoved is a log parse operation binding the contract event 0xa131d16963736e4c641f27a7f82f2e350b5971e555ae06ae906892bbba0a0939.
//
// Solidity: event StakeRemoved(address indexed user, uint256 amount, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) ParseStakeRemoved(log types.Log) (*StakingStakeRemoved, error) {
	event := new(StakingStakeRemoved)
	if err := _Staking.contract.UnpackLog(event, "StakeRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakingStakeTransferredIterator is returned from FilterStakeTransferred and is used to iterate over the raw logs and unpacked data for StakeTransferred events raised by the Staking contract.
type StakingStakeTransferredIterator struct {
	Event *StakingStakeTransferred // Event containing the contract specifics and raw log

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
func (it *StakingStakeTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakingStakeTransferred)
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
		it.Event = new(StakingStakeTransferred)
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
func (it *StakingStakeTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakingStakeTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakingStakeTransferred represents a StakeTransferred event raised by the Staking contract.
type StakingStakeTransferred struct {
	From          common.Address
	Amount        *big.Int
	To            common.Address
	VegaPublicKey [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterStakeTransferred is a free log retrieval operation binding the contract event 0x296aca09e6f616abedcd9cd45ac378207310452b7a713289374fd1b35e2c2fbe.
//
// Solidity: event Stake_Transferred(address indexed from, uint256 amount, address indexed to, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) FilterStakeTransferred(opts *bind.FilterOpts, from []common.Address, to []common.Address, vega_public_key [][32]byte) (*StakingStakeTransferredIterator, error) {
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var vega_public_keyRule []interface{}
	for _, vega_public_keyItem := range vega_public_key {
		vega_public_keyRule = append(vega_public_keyRule, vega_public_keyItem)
	}

	logs, sub, err := _Staking.contract.FilterLogs(opts, "Stake_Transferred", fromRule, toRule, vega_public_keyRule)
	if err != nil {
		return nil, err
	}
	return &StakingStakeTransferredIterator{contract: _Staking.contract, event: "Stake_Transferred", logs: logs, sub: sub}, nil
}

// WatchStakeTransferred is a free log subscription operation binding the contract event 0x296aca09e6f616abedcd9cd45ac378207310452b7a713289374fd1b35e2c2fbe.
//
// Solidity: event Stake_Transferred(address indexed from, uint256 amount, address indexed to, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) WatchStakeTransferred(opts *bind.WatchOpts, sink chan<- *StakingStakeTransferred, from []common.Address, to []common.Address, vega_public_key [][32]byte) (event.Subscription, error) {
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var vega_public_keyRule []interface{}
	for _, vega_public_keyItem := range vega_public_key {
		vega_public_keyRule = append(vega_public_keyRule, vega_public_keyItem)
	}

	logs, sub, err := _Staking.contract.WatchLogs(opts, "Stake_Transferred", fromRule, toRule, vega_public_keyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakingStakeTransferred)
				if err := _Staking.contract.UnpackLog(event, "Stake_Transferred", log); err != nil {
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

// ParseStakeTransferred is a log parse operation binding the contract event 0x296aca09e6f616abedcd9cd45ac378207310452b7a713289374fd1b35e2c2fbe.
//
// Solidity: event Stake_Transferred(address indexed from, uint256 amount, address indexed to, bytes32 indexed vega_public_key)
func (_Staking *StakingFilterer) ParseStakeTransferred(log types.Log) (*StakingStakeTransferred, error) {
	event := new(StakingStakeTransferred)
	if err := _Staking.contract.UnpackLog(event, "Stake_Transferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
