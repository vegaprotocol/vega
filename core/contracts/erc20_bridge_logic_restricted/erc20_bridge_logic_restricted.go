// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package erc20_bridge_logic_restricted

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

// Erc20BridgeLogicRestrictedMetaData contains all meta data concerning the Erc20BridgeLogicRestricted contract.
var Erc20BridgeLogicRestrictedMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"addresspayable\",\"name\":\"erc20_asset_pool\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user_address\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"Asset_Deposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"lifetime_limit\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdraw_threshold\",\"type\":\"uint256\"}],\"name\":\"Asset_Limits_Updated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"vega_asset_id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Listed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Removed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user_address\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Withdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Bridge_Resumed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Bridge_Stopped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"withdraw_delay\",\"type\":\"uint256\"}],\"name\":\"Bridge_Withdraw_Delay_Set\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"depositor\",\"type\":\"address\"}],\"name\":\"Depositor_Exempted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"depositor\",\"type\":\"address\"}],\"name\":\"Depositor_Exemption_Revoked\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"default_withdraw_delay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"deposit_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"erc20_asset_pool_address\",\"outputs\":[{\"internalType\":\"addresspayable\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"exempt_depositor\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"get_asset_deposit_lifetime_limit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vega_asset_id\",\"type\":\"bytes32\"}],\"name\":\"get_asset_source\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_multisig_control_address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"get_vega_asset_id\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"get_withdraw_threshold\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"global_resume\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"global_stop\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"is_asset_listed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"depositor\",\"type\":\"address\"}],\"name\":\"is_exempt_depositor\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"is_stopped\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"vega_asset_id\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"lifetime_limit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"withdraw_threshold\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"list_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"remove_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"revoke_exempt_depositor\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"lifetime_limit\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"threshold\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"set_asset_limits\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"delay\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"set_withdraw_delay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"creation\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"withdraw_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// Erc20BridgeLogicRestrictedABI is the input ABI used to generate the binding from.
// Deprecated: Use Erc20BridgeLogicRestrictedMetaData.ABI instead.
var Erc20BridgeLogicRestrictedABI = Erc20BridgeLogicRestrictedMetaData.ABI

// Erc20BridgeLogicRestricted is an auto generated Go binding around an Ethereum contract.
type Erc20BridgeLogicRestricted struct {
	Erc20BridgeLogicRestrictedCaller     // Read-only binding to the contract
	Erc20BridgeLogicRestrictedTransactor // Write-only binding to the contract
	Erc20BridgeLogicRestrictedFilterer   // Log filterer for contract events
}

// Erc20BridgeLogicRestrictedCaller is an auto generated read-only Go binding around an Ethereum contract.
type Erc20BridgeLogicRestrictedCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc20BridgeLogicRestrictedTransactor is an auto generated write-only Go binding around an Ethereum contract.
type Erc20BridgeLogicRestrictedTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc20BridgeLogicRestrictedFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Erc20BridgeLogicRestrictedFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc20BridgeLogicRestrictedSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Erc20BridgeLogicRestrictedSession struct {
	Contract     *Erc20BridgeLogicRestricted // Generic contract binding to set the session for
	CallOpts     bind.CallOpts               // Call options to use throughout this session
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// Erc20BridgeLogicRestrictedCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Erc20BridgeLogicRestrictedCallerSession struct {
	Contract *Erc20BridgeLogicRestrictedCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                     // Call options to use throughout this session
}

// Erc20BridgeLogicRestrictedTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Erc20BridgeLogicRestrictedTransactorSession struct {
	Contract     *Erc20BridgeLogicRestrictedTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                     // Transaction auth options to use throughout this session
}

// Erc20BridgeLogicRestrictedRaw is an auto generated low-level Go binding around an Ethereum contract.
type Erc20BridgeLogicRestrictedRaw struct {
	Contract *Erc20BridgeLogicRestricted // Generic contract binding to access the raw methods on
}

// Erc20BridgeLogicRestrictedCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Erc20BridgeLogicRestrictedCallerRaw struct {
	Contract *Erc20BridgeLogicRestrictedCaller // Generic read-only contract binding to access the raw methods on
}

// Erc20BridgeLogicRestrictedTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Erc20BridgeLogicRestrictedTransactorRaw struct {
	Contract *Erc20BridgeLogicRestrictedTransactor // Generic write-only contract binding to access the raw methods on
}

// NewErc20BridgeLogicRestricted creates a new instance of Erc20BridgeLogicRestricted, bound to a specific deployed contract.
func NewErc20BridgeLogicRestricted(address common.Address, backend bind.ContractBackend) (*Erc20BridgeLogicRestricted, error) {
	contract, err := bindErc20BridgeLogicRestricted(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestricted{Erc20BridgeLogicRestrictedCaller: Erc20BridgeLogicRestrictedCaller{contract: contract}, Erc20BridgeLogicRestrictedTransactor: Erc20BridgeLogicRestrictedTransactor{contract: contract}, Erc20BridgeLogicRestrictedFilterer: Erc20BridgeLogicRestrictedFilterer{contract: contract}}, nil
}

// NewErc20BridgeLogicRestrictedCaller creates a new read-only instance of Erc20BridgeLogicRestricted, bound to a specific deployed contract.
func NewErc20BridgeLogicRestrictedCaller(address common.Address, caller bind.ContractCaller) (*Erc20BridgeLogicRestrictedCaller, error) {
	contract, err := bindErc20BridgeLogicRestricted(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedCaller{contract: contract}, nil
}

// NewErc20BridgeLogicRestrictedTransactor creates a new write-only instance of Erc20BridgeLogicRestricted, bound to a specific deployed contract.
func NewErc20BridgeLogicRestrictedTransactor(address common.Address, transactor bind.ContractTransactor) (*Erc20BridgeLogicRestrictedTransactor, error) {
	contract, err := bindErc20BridgeLogicRestricted(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedTransactor{contract: contract}, nil
}

// NewErc20BridgeLogicRestrictedFilterer creates a new log filterer instance of Erc20BridgeLogicRestricted, bound to a specific deployed contract.
func NewErc20BridgeLogicRestrictedFilterer(address common.Address, filterer bind.ContractFilterer) (*Erc20BridgeLogicRestrictedFilterer, error) {
	contract, err := bindErc20BridgeLogicRestricted(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedFilterer{contract: contract}, nil
}

// bindErc20BridgeLogicRestricted binds a generic wrapper to an already deployed contract.
func bindErc20BridgeLogicRestricted(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Erc20BridgeLogicRestrictedMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc20BridgeLogicRestricted.Contract.Erc20BridgeLogicRestrictedCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.Erc20BridgeLogicRestrictedTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.Erc20BridgeLogicRestrictedTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc20BridgeLogicRestricted.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.contract.Transact(opts, method, params...)
}

// DefaultWithdrawDelay is a free data retrieval call binding the contract method 0x3f4f199d.
//
// Solidity: function default_withdraw_delay() view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) DefaultWithdrawDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "default_withdraw_delay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DefaultWithdrawDelay is a free data retrieval call binding the contract method 0x3f4f199d.
//
// Solidity: function default_withdraw_delay() view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) DefaultWithdrawDelay() (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.DefaultWithdrawDelay(&_Erc20BridgeLogicRestricted.CallOpts)
}

// DefaultWithdrawDelay is a free data retrieval call binding the contract method 0x3f4f199d.
//
// Solidity: function default_withdraw_delay() view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) DefaultWithdrawDelay() (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.DefaultWithdrawDelay(&_Erc20BridgeLogicRestricted.CallOpts)
}

// Erc20AssetPoolAddress is a free data retrieval call binding the contract method 0x9356aab8.
//
// Solidity: function erc20_asset_pool_address() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) Erc20AssetPoolAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "erc20_asset_pool_address")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Erc20AssetPoolAddress is a free data retrieval call binding the contract method 0x9356aab8.
//
// Solidity: function erc20_asset_pool_address() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) Erc20AssetPoolAddress() (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.Erc20AssetPoolAddress(&_Erc20BridgeLogicRestricted.CallOpts)
}

// Erc20AssetPoolAddress is a free data retrieval call binding the contract method 0x9356aab8.
//
// Solidity: function erc20_asset_pool_address() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) Erc20AssetPoolAddress() (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.Erc20AssetPoolAddress(&_Erc20BridgeLogicRestricted.CallOpts)
}

// GetAssetDepositLifetimeLimit is a free data retrieval call binding the contract method 0x354a897a.
//
// Solidity: function get_asset_deposit_lifetime_limit(address asset_source) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetAssetDepositLifetimeLimit(opts *bind.CallOpts, asset_source common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "get_asset_deposit_lifetime_limit", asset_source)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAssetDepositLifetimeLimit is a free data retrieval call binding the contract method 0x354a897a.
//
// Solidity: function get_asset_deposit_lifetime_limit(address asset_source) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetAssetDepositLifetimeLimit(asset_source common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetDepositLifetimeLimit(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// GetAssetDepositLifetimeLimit is a free data retrieval call binding the contract method 0x354a897a.
//
// Solidity: function get_asset_deposit_lifetime_limit(address asset_source) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetAssetDepositLifetimeLimit(asset_source common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetDepositLifetimeLimit(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// GetAssetSource is a free data retrieval call binding the contract method 0x786b0bc0.
//
// Solidity: function get_asset_source(bytes32 vega_asset_id) view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetAssetSource(opts *bind.CallOpts, vega_asset_id [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "get_asset_source", vega_asset_id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAssetSource is a free data retrieval call binding the contract method 0x786b0bc0.
//
// Solidity: function get_asset_source(bytes32 vega_asset_id) view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetAssetSource(vega_asset_id [32]byte) (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetSource(&_Erc20BridgeLogicRestricted.CallOpts, vega_asset_id)
}

// GetAssetSource is a free data retrieval call binding the contract method 0x786b0bc0.
//
// Solidity: function get_asset_source(bytes32 vega_asset_id) view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetAssetSource(vega_asset_id [32]byte) (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetSource(&_Erc20BridgeLogicRestricted.CallOpts, vega_asset_id)
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0xc58dc3b9.
//
// Solidity: function get_multisig_control_address() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetMultisigControlAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "get_multisig_control_address")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0xc58dc3b9.
//
// Solidity: function get_multisig_control_address() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetMultisigControlAddress() (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetMultisigControlAddress(&_Erc20BridgeLogicRestricted.CallOpts)
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0xc58dc3b9.
//
// Solidity: function get_multisig_control_address() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetMultisigControlAddress() (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetMultisigControlAddress(&_Erc20BridgeLogicRestricted.CallOpts)
}

// GetVegaAssetId is a free data retrieval call binding the contract method 0xa06b5d39.
//
// Solidity: function get_vega_asset_id(address asset_source) view returns(bytes32)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetVegaAssetId(opts *bind.CallOpts, asset_source common.Address) ([32]byte, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "get_vega_asset_id", asset_source)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetVegaAssetId is a free data retrieval call binding the contract method 0xa06b5d39.
//
// Solidity: function get_vega_asset_id(address asset_source) view returns(bytes32)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetVegaAssetId(asset_source common.Address) ([32]byte, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetVegaAssetId(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// GetVegaAssetId is a free data retrieval call binding the contract method 0xa06b5d39.
//
// Solidity: function get_vega_asset_id(address asset_source) view returns(bytes32)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetVegaAssetId(asset_source common.Address) ([32]byte, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetVegaAssetId(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// GetWithdrawThreshold is a free data retrieval call binding the contract method 0xe8a7bce0.
//
// Solidity: function get_withdraw_threshold(address asset_source) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetWithdrawThreshold(opts *bind.CallOpts, asset_source common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "get_withdraw_threshold", asset_source)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWithdrawThreshold is a free data retrieval call binding the contract method 0xe8a7bce0.
//
// Solidity: function get_withdraw_threshold(address asset_source) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetWithdrawThreshold(asset_source common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetWithdrawThreshold(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// GetWithdrawThreshold is a free data retrieval call binding the contract method 0xe8a7bce0.
//
// Solidity: function get_withdraw_threshold(address asset_source) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetWithdrawThreshold(asset_source common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetWithdrawThreshold(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// IsAssetListed is a free data retrieval call binding the contract method 0x7fd27b7f.
//
// Solidity: function is_asset_listed(address asset_source) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) IsAssetListed(opts *bind.CallOpts, asset_source common.Address) (bool, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "is_asset_listed", asset_source)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAssetListed is a free data retrieval call binding the contract method 0x7fd27b7f.
//
// Solidity: function is_asset_listed(address asset_source) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) IsAssetListed(asset_source common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsAssetListed(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// IsAssetListed is a free data retrieval call binding the contract method 0x7fd27b7f.
//
// Solidity: function is_asset_listed(address asset_source) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) IsAssetListed(asset_source common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsAssetListed(&_Erc20BridgeLogicRestricted.CallOpts, asset_source)
}

// IsExemptDepositor is a free data retrieval call binding the contract method 0x15c0df9d.
//
// Solidity: function is_exempt_depositor(address depositor) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) IsExemptDepositor(opts *bind.CallOpts, depositor common.Address) (bool, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "is_exempt_depositor", depositor)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsExemptDepositor is a free data retrieval call binding the contract method 0x15c0df9d.
//
// Solidity: function is_exempt_depositor(address depositor) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) IsExemptDepositor(depositor common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsExemptDepositor(&_Erc20BridgeLogicRestricted.CallOpts, depositor)
}

// IsExemptDepositor is a free data retrieval call binding the contract method 0x15c0df9d.
//
// Solidity: function is_exempt_depositor(address depositor) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) IsExemptDepositor(depositor common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsExemptDepositor(&_Erc20BridgeLogicRestricted.CallOpts, depositor)
}

// IsStopped is a free data retrieval call binding the contract method 0xe272e9d0.
//
// Solidity: function is_stopped() view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) IsStopped(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "is_stopped")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsStopped is a free data retrieval call binding the contract method 0xe272e9d0.
//
// Solidity: function is_stopped() view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) IsStopped() (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsStopped(&_Erc20BridgeLogicRestricted.CallOpts)
}

// IsStopped is a free data retrieval call binding the contract method 0xe272e9d0.
//
// Solidity: function is_stopped() view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) IsStopped() (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsStopped(&_Erc20BridgeLogicRestricted.CallOpts)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf7683932.
//
// Solidity: function deposit_asset(address asset_source, uint256 amount, bytes32 vega_public_key) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) DepositAsset(opts *bind.TransactOpts, asset_source common.Address, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "deposit_asset", asset_source, amount, vega_public_key)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf7683932.
//
// Solidity: function deposit_asset(address asset_source, uint256 amount, bytes32 vega_public_key) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) DepositAsset(asset_source common.Address, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.DepositAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, amount, vega_public_key)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf7683932.
//
// Solidity: function deposit_asset(address asset_source, uint256 amount, bytes32 vega_public_key) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) DepositAsset(asset_source common.Address, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.DepositAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, amount, vega_public_key)
}

// ExemptDepositor is a paid mutator transaction binding the contract method 0xb76fbb75.
//
// Solidity: function exempt_depositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) ExemptDepositor(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "exempt_depositor")
}

// ExemptDepositor is a paid mutator transaction binding the contract method 0xb76fbb75.
//
// Solidity: function exempt_depositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) ExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// ExemptDepositor is a paid mutator transaction binding the contract method 0xb76fbb75.
//
// Solidity: function exempt_depositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) ExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// GlobalResume is a paid mutator transaction binding the contract method 0xd72ed529.
//
// Solidity: function global_resume(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) GlobalResume(opts *bind.TransactOpts, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "global_resume", nonce, signatures)
}

// GlobalResume is a paid mutator transaction binding the contract method 0xd72ed529.
//
// Solidity: function global_resume(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GlobalResume(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalResume(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// GlobalResume is a paid mutator transaction binding the contract method 0xd72ed529.
//
// Solidity: function global_resume(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) GlobalResume(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalResume(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// GlobalStop is a paid mutator transaction binding the contract method 0x9dfd3c88.
//
// Solidity: function global_stop(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) GlobalStop(opts *bind.TransactOpts, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "global_stop", nonce, signatures)
}

// GlobalStop is a paid mutator transaction binding the contract method 0x9dfd3c88.
//
// Solidity: function global_stop(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GlobalStop(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalStop(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// GlobalStop is a paid mutator transaction binding the contract method 0x9dfd3c88.
//
// Solidity: function global_stop(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) GlobalStop(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalStop(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x0ff3562c.
//
// Solidity: function list_asset(address asset_source, bytes32 vega_asset_id, uint256 lifetime_limit, uint256 withdraw_threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) ListAsset(opts *bind.TransactOpts, asset_source common.Address, vega_asset_id [32]byte, lifetime_limit *big.Int, withdraw_threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "list_asset", asset_source, vega_asset_id, lifetime_limit, withdraw_threshold, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x0ff3562c.
//
// Solidity: function list_asset(address asset_source, bytes32 vega_asset_id, uint256 lifetime_limit, uint256 withdraw_threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) ListAsset(asset_source common.Address, vega_asset_id [32]byte, lifetime_limit *big.Int, withdraw_threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ListAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, vega_asset_id, lifetime_limit, withdraw_threshold, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x0ff3562c.
//
// Solidity: function list_asset(address asset_source, bytes32 vega_asset_id, uint256 lifetime_limit, uint256 withdraw_threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) ListAsset(asset_source common.Address, vega_asset_id [32]byte, lifetime_limit *big.Int, withdraw_threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ListAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, vega_asset_id, lifetime_limit, withdraw_threshold, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc76de358.
//
// Solidity: function remove_asset(address asset_source, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) RemoveAsset(opts *bind.TransactOpts, asset_source common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "remove_asset", asset_source, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc76de358.
//
// Solidity: function remove_asset(address asset_source, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) RemoveAsset(asset_source common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RemoveAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc76de358.
//
// Solidity: function remove_asset(address asset_source, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) RemoveAsset(asset_source common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RemoveAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, nonce, signatures)
}

// RevokeExemptDepositor is a paid mutator transaction binding the contract method 0x6a1c6fa4.
//
// Solidity: function revoke_exempt_depositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) RevokeExemptDepositor(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "revoke_exempt_depositor")
}

// RevokeExemptDepositor is a paid mutator transaction binding the contract method 0x6a1c6fa4.
//
// Solidity: function revoke_exempt_depositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) RevokeExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RevokeExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// RevokeExemptDepositor is a paid mutator transaction binding the contract method 0x6a1c6fa4.
//
// Solidity: function revoke_exempt_depositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) RevokeExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RevokeExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// SetAssetLimits is a paid mutator transaction binding the contract method 0x41fb776d.
//
// Solidity: function set_asset_limits(address asset_source, uint256 lifetime_limit, uint256 threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) SetAssetLimits(opts *bind.TransactOpts, asset_source common.Address, lifetime_limit *big.Int, threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "set_asset_limits", asset_source, lifetime_limit, threshold, nonce, signatures)
}

// SetAssetLimits is a paid mutator transaction binding the contract method 0x41fb776d.
//
// Solidity: function set_asset_limits(address asset_source, uint256 lifetime_limit, uint256 threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) SetAssetLimits(asset_source common.Address, lifetime_limit *big.Int, threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetAssetLimits(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, lifetime_limit, threshold, nonce, signatures)
}

// SetAssetLimits is a paid mutator transaction binding the contract method 0x41fb776d.
//
// Solidity: function set_asset_limits(address asset_source, uint256 lifetime_limit, uint256 threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) SetAssetLimits(asset_source common.Address, lifetime_limit *big.Int, threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetAssetLimits(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, lifetime_limit, threshold, nonce, signatures)
}

// SetWithdrawDelay is a paid mutator transaction binding the contract method 0x5a246728.
//
// Solidity: function set_withdraw_delay(uint256 delay, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) SetWithdrawDelay(opts *bind.TransactOpts, delay *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "set_withdraw_delay", delay, nonce, signatures)
}

// SetWithdrawDelay is a paid mutator transaction binding the contract method 0x5a246728.
//
// Solidity: function set_withdraw_delay(uint256 delay, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) SetWithdrawDelay(delay *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetWithdrawDelay(&_Erc20BridgeLogicRestricted.TransactOpts, delay, nonce, signatures)
}

// SetWithdrawDelay is a paid mutator transaction binding the contract method 0x5a246728.
//
// Solidity: function set_withdraw_delay(uint256 delay, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) SetWithdrawDelay(delay *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetWithdrawDelay(&_Erc20BridgeLogicRestricted.TransactOpts, delay, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x3ad90635.
//
// Solidity: function withdraw_asset(address asset_source, uint256 amount, address target, uint256 creation, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) WithdrawAsset(opts *bind.TransactOpts, asset_source common.Address, amount *big.Int, target common.Address, creation *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "withdraw_asset", asset_source, amount, target, creation, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x3ad90635.
//
// Solidity: function withdraw_asset(address asset_source, uint256 amount, address target, uint256 creation, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) WithdrawAsset(asset_source common.Address, amount *big.Int, target common.Address, creation *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.WithdrawAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, amount, target, creation, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x3ad90635.
//
// Solidity: function withdraw_asset(address asset_source, uint256 amount, address target, uint256 creation, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) WithdrawAsset(asset_source common.Address, amount *big.Int, target common.Address, creation *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.WithdrawAsset(&_Erc20BridgeLogicRestricted.TransactOpts, asset_source, amount, target, creation, nonce, signatures)
}

// Erc20BridgeLogicRestrictedAssetDepositedIterator is returned from FilterAssetDeposited and is used to iterate over the raw logs and unpacked data for AssetDeposited events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetDepositedIterator struct {
	Event *Erc20BridgeLogicRestrictedAssetDeposited // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedAssetDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedAssetDeposited)
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
		it.Event = new(Erc20BridgeLogicRestrictedAssetDeposited)
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
func (it *Erc20BridgeLogicRestrictedAssetDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedAssetDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedAssetDeposited represents a AssetDeposited event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetDeposited struct {
	UserAddress   common.Address
	AssetSource   common.Address
	Amount        *big.Int
	VegaPublicKey [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAssetDeposited is a free log retrieval operation binding the contract event 0x3724ff5e82ddc640a08d68b0b782a5991aea0de51a8dd10a59cdbe5b3ec4e6bf.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 amount, bytes32 vega_public_key)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetDeposited(opts *bind.FilterOpts, user_address []common.Address, asset_source []common.Address) (*Erc20BridgeLogicRestrictedAssetDepositedIterator, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Asset_Deposited", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetDepositedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Asset_Deposited", logs: logs, sub: sub}, nil
}

// WatchAssetDeposited is a free log subscription operation binding the contract event 0x3724ff5e82ddc640a08d68b0b782a5991aea0de51a8dd10a59cdbe5b3ec4e6bf.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 amount, bytes32 vega_public_key)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetDeposited(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetDeposited, user_address []common.Address, asset_source []common.Address) (event.Subscription, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Asset_Deposited", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedAssetDeposited)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Deposited", log); err != nil {
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

// ParseAssetDeposited is a log parse operation binding the contract event 0x3724ff5e82ddc640a08d68b0b782a5991aea0de51a8dd10a59cdbe5b3ec4e6bf.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 amount, bytes32 vega_public_key)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetDeposited(log types.Log) (*Erc20BridgeLogicRestrictedAssetDeposited, error) {
	event := new(Erc20BridgeLogicRestrictedAssetDeposited)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Deposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator is returned from FilterAssetLimitsUpdated and is used to iterate over the raw logs and unpacked data for AssetLimitsUpdated events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator struct {
	Event *Erc20BridgeLogicRestrictedAssetLimitsUpdated // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedAssetLimitsUpdated)
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
		it.Event = new(Erc20BridgeLogicRestrictedAssetLimitsUpdated)
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
func (it *Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedAssetLimitsUpdated represents a AssetLimitsUpdated event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetLimitsUpdated struct {
	AssetSource       common.Address
	LifetimeLimit     *big.Int
	WithdrawThreshold *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterAssetLimitsUpdated is a free log retrieval operation binding the contract event 0xfc7eab762b8751ad85c101fd1025c763b4e8d48f2093f506629b606618e884fe.
//
// Solidity: event Asset_Limits_Updated(address indexed asset_source, uint256 lifetime_limit, uint256 withdraw_threshold)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetLimitsUpdated(opts *bind.FilterOpts, asset_source []common.Address) (*Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Asset_Limits_Updated", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Asset_Limits_Updated", logs: logs, sub: sub}, nil
}

// WatchAssetLimitsUpdated is a free log subscription operation binding the contract event 0xfc7eab762b8751ad85c101fd1025c763b4e8d48f2093f506629b606618e884fe.
//
// Solidity: event Asset_Limits_Updated(address indexed asset_source, uint256 lifetime_limit, uint256 withdraw_threshold)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetLimitsUpdated(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetLimitsUpdated, asset_source []common.Address) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Asset_Limits_Updated", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedAssetLimitsUpdated)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Limits_Updated", log); err != nil {
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

// ParseAssetLimitsUpdated is a log parse operation binding the contract event 0xfc7eab762b8751ad85c101fd1025c763b4e8d48f2093f506629b606618e884fe.
//
// Solidity: event Asset_Limits_Updated(address indexed asset_source, uint256 lifetime_limit, uint256 withdraw_threshold)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetLimitsUpdated(log types.Log) (*Erc20BridgeLogicRestrictedAssetLimitsUpdated, error) {
	event := new(Erc20BridgeLogicRestrictedAssetLimitsUpdated)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Limits_Updated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedAssetListedIterator is returned from FilterAssetListed and is used to iterate over the raw logs and unpacked data for AssetListed events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetListedIterator struct {
	Event *Erc20BridgeLogicRestrictedAssetListed // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedAssetListedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedAssetListed)
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
		it.Event = new(Erc20BridgeLogicRestrictedAssetListed)
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
func (it *Erc20BridgeLogicRestrictedAssetListedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedAssetListedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedAssetListed represents a AssetListed event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetListed struct {
	AssetSource common.Address
	VegaAssetId [32]byte
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetListed is a free log retrieval operation binding the contract event 0x4180d77d05ff0d31650c548c23f2de07a3da3ad42e3dd6edd817b438a150452e.
//
// Solidity: event Asset_Listed(address indexed asset_source, bytes32 indexed vega_asset_id, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetListed(opts *bind.FilterOpts, asset_source []common.Address, vega_asset_id [][32]byte) (*Erc20BridgeLogicRestrictedAssetListedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var vega_asset_idRule []interface{}
	for _, vega_asset_idItem := range vega_asset_id {
		vega_asset_idRule = append(vega_asset_idRule, vega_asset_idItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Asset_Listed", asset_sourceRule, vega_asset_idRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetListedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Asset_Listed", logs: logs, sub: sub}, nil
}

// WatchAssetListed is a free log subscription operation binding the contract event 0x4180d77d05ff0d31650c548c23f2de07a3da3ad42e3dd6edd817b438a150452e.
//
// Solidity: event Asset_Listed(address indexed asset_source, bytes32 indexed vega_asset_id, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetListed(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetListed, asset_source []common.Address, vega_asset_id [][32]byte) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var vega_asset_idRule []interface{}
	for _, vega_asset_idItem := range vega_asset_id {
		vega_asset_idRule = append(vega_asset_idRule, vega_asset_idItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Asset_Listed", asset_sourceRule, vega_asset_idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedAssetListed)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Listed", log); err != nil {
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

// ParseAssetListed is a log parse operation binding the contract event 0x4180d77d05ff0d31650c548c23f2de07a3da3ad42e3dd6edd817b438a150452e.
//
// Solidity: event Asset_Listed(address indexed asset_source, bytes32 indexed vega_asset_id, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetListed(log types.Log) (*Erc20BridgeLogicRestrictedAssetListed, error) {
	event := new(Erc20BridgeLogicRestrictedAssetListed)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Listed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedAssetRemovedIterator is returned from FilterAssetRemoved and is used to iterate over the raw logs and unpacked data for AssetRemoved events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetRemovedIterator struct {
	Event *Erc20BridgeLogicRestrictedAssetRemoved // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedAssetRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedAssetRemoved)
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
		it.Event = new(Erc20BridgeLogicRestrictedAssetRemoved)
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
func (it *Erc20BridgeLogicRestrictedAssetRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedAssetRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedAssetRemoved represents a AssetRemoved event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetRemoved struct {
	AssetSource common.Address
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetRemoved is a free log retrieval operation binding the contract event 0x58ad5e799e2df93ab408be0e5c1870d44c80b5bca99dfaf7ddf0dab5e6b155c9.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetRemoved(opts *bind.FilterOpts, asset_source []common.Address) (*Erc20BridgeLogicRestrictedAssetRemovedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Asset_Removed", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetRemovedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Asset_Removed", logs: logs, sub: sub}, nil
}

// WatchAssetRemoved is a free log subscription operation binding the contract event 0x58ad5e799e2df93ab408be0e5c1870d44c80b5bca99dfaf7ddf0dab5e6b155c9.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetRemoved(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetRemoved, asset_source []common.Address) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Asset_Removed", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedAssetRemoved)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Removed", log); err != nil {
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

// ParseAssetRemoved is a log parse operation binding the contract event 0x58ad5e799e2df93ab408be0e5c1870d44c80b5bca99dfaf7ddf0dab5e6b155c9.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetRemoved(log types.Log) (*Erc20BridgeLogicRestrictedAssetRemoved, error) {
	event := new(Erc20BridgeLogicRestrictedAssetRemoved)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Removed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedAssetWithdrawnIterator is returned from FilterAssetWithdrawn and is used to iterate over the raw logs and unpacked data for AssetWithdrawn events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetWithdrawnIterator struct {
	Event *Erc20BridgeLogicRestrictedAssetWithdrawn // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedAssetWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedAssetWithdrawn)
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
		it.Event = new(Erc20BridgeLogicRestrictedAssetWithdrawn)
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
func (it *Erc20BridgeLogicRestrictedAssetWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedAssetWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedAssetWithdrawn represents a AssetWithdrawn event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedAssetWithdrawn struct {
	UserAddress common.Address
	AssetSource common.Address
	Amount      *big.Int
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetWithdrawn is a free log retrieval operation binding the contract event 0xa79be4f3361e32d396d64c478ecef73732cb40b2a75702c3b3b3226a2c83b5df.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 amount, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetWithdrawn(opts *bind.FilterOpts, user_address []common.Address, asset_source []common.Address) (*Erc20BridgeLogicRestrictedAssetWithdrawnIterator, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Asset_Withdrawn", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetWithdrawnIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Asset_Withdrawn", logs: logs, sub: sub}, nil
}

// WatchAssetWithdrawn is a free log subscription operation binding the contract event 0xa79be4f3361e32d396d64c478ecef73732cb40b2a75702c3b3b3226a2c83b5df.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 amount, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetWithdrawn(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetWithdrawn, user_address []common.Address, asset_source []common.Address) (event.Subscription, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Asset_Withdrawn", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedAssetWithdrawn)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Withdrawn", log); err != nil {
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

// ParseAssetWithdrawn is a log parse operation binding the contract event 0xa79be4f3361e32d396d64c478ecef73732cb40b2a75702c3b3b3226a2c83b5df.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 amount, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetWithdrawn(log types.Log) (*Erc20BridgeLogicRestrictedAssetWithdrawn, error) {
	event := new(Erc20BridgeLogicRestrictedAssetWithdrawn)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Asset_Withdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedBridgeResumedIterator is returned from FilterBridgeResumed and is used to iterate over the raw logs and unpacked data for BridgeResumed events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedBridgeResumedIterator struct {
	Event *Erc20BridgeLogicRestrictedBridgeResumed // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedBridgeResumedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedBridgeResumed)
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
		it.Event = new(Erc20BridgeLogicRestrictedBridgeResumed)
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
func (it *Erc20BridgeLogicRestrictedBridgeResumedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedBridgeResumedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedBridgeResumed represents a BridgeResumed event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedBridgeResumed struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterBridgeResumed is a free log retrieval operation binding the contract event 0x79c02b0e60e0f00fe0370791204f2f175fe3f06f4816f3506ad4fa1b8e8cde0f.
//
// Solidity: event Bridge_Resumed()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterBridgeResumed(opts *bind.FilterOpts) (*Erc20BridgeLogicRestrictedBridgeResumedIterator, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Bridge_Resumed")
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedBridgeResumedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Bridge_Resumed", logs: logs, sub: sub}, nil
}

// WatchBridgeResumed is a free log subscription operation binding the contract event 0x79c02b0e60e0f00fe0370791204f2f175fe3f06f4816f3506ad4fa1b8e8cde0f.
//
// Solidity: event Bridge_Resumed()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchBridgeResumed(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedBridgeResumed) (event.Subscription, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Bridge_Resumed")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedBridgeResumed)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Bridge_Resumed", log); err != nil {
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

// ParseBridgeResumed is a log parse operation binding the contract event 0x79c02b0e60e0f00fe0370791204f2f175fe3f06f4816f3506ad4fa1b8e8cde0f.
//
// Solidity: event Bridge_Resumed()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseBridgeResumed(log types.Log) (*Erc20BridgeLogicRestrictedBridgeResumed, error) {
	event := new(Erc20BridgeLogicRestrictedBridgeResumed)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Bridge_Resumed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedBridgeStoppedIterator is returned from FilterBridgeStopped and is used to iterate over the raw logs and unpacked data for BridgeStopped events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedBridgeStoppedIterator struct {
	Event *Erc20BridgeLogicRestrictedBridgeStopped // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedBridgeStoppedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedBridgeStopped)
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
		it.Event = new(Erc20BridgeLogicRestrictedBridgeStopped)
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
func (it *Erc20BridgeLogicRestrictedBridgeStoppedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedBridgeStoppedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedBridgeStopped represents a BridgeStopped event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedBridgeStopped struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterBridgeStopped is a free log retrieval operation binding the contract event 0x129d99581c8e70519df1f0733d3212f33d0ed3ea6144adacc336c647f1d36382.
//
// Solidity: event Bridge_Stopped()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterBridgeStopped(opts *bind.FilterOpts) (*Erc20BridgeLogicRestrictedBridgeStoppedIterator, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Bridge_Stopped")
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedBridgeStoppedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Bridge_Stopped", logs: logs, sub: sub}, nil
}

// WatchBridgeStopped is a free log subscription operation binding the contract event 0x129d99581c8e70519df1f0733d3212f33d0ed3ea6144adacc336c647f1d36382.
//
// Solidity: event Bridge_Stopped()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchBridgeStopped(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedBridgeStopped) (event.Subscription, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Bridge_Stopped")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedBridgeStopped)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Bridge_Stopped", log); err != nil {
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

// ParseBridgeStopped is a log parse operation binding the contract event 0x129d99581c8e70519df1f0733d3212f33d0ed3ea6144adacc336c647f1d36382.
//
// Solidity: event Bridge_Stopped()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseBridgeStopped(log types.Log) (*Erc20BridgeLogicRestrictedBridgeStopped, error) {
	event := new(Erc20BridgeLogicRestrictedBridgeStopped)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Bridge_Stopped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator is returned from FilterBridgeWithdrawDelaySet and is used to iterate over the raw logs and unpacked data for BridgeWithdrawDelaySet events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator struct {
	Event *Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet)
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
		it.Event = new(Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet)
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
func (it *Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet represents a BridgeWithdrawDelaySet event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet struct {
	WithdrawDelay *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterBridgeWithdrawDelaySet is a free log retrieval operation binding the contract event 0x1c7e8f73a01b8af4e18dd34455a42a45ad742bdb79cfda77bbdf50db2391fc88.
//
// Solidity: event Bridge_Withdraw_Delay_Set(uint256 withdraw_delay)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterBridgeWithdrawDelaySet(opts *bind.FilterOpts) (*Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Bridge_Withdraw_Delay_Set")
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Bridge_Withdraw_Delay_Set", logs: logs, sub: sub}, nil
}

// WatchBridgeWithdrawDelaySet is a free log subscription operation binding the contract event 0x1c7e8f73a01b8af4e18dd34455a42a45ad742bdb79cfda77bbdf50db2391fc88.
//
// Solidity: event Bridge_Withdraw_Delay_Set(uint256 withdraw_delay)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchBridgeWithdrawDelaySet(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet) (event.Subscription, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Bridge_Withdraw_Delay_Set")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Bridge_Withdraw_Delay_Set", log); err != nil {
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

// ParseBridgeWithdrawDelaySet is a log parse operation binding the contract event 0x1c7e8f73a01b8af4e18dd34455a42a45ad742bdb79cfda77bbdf50db2391fc88.
//
// Solidity: event Bridge_Withdraw_Delay_Set(uint256 withdraw_delay)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseBridgeWithdrawDelaySet(log types.Log) (*Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet, error) {
	event := new(Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Bridge_Withdraw_Delay_Set", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedDepositorExemptedIterator is returned from FilterDepositorExempted and is used to iterate over the raw logs and unpacked data for DepositorExempted events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedDepositorExemptedIterator struct {
	Event *Erc20BridgeLogicRestrictedDepositorExempted // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedDepositorExemptedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedDepositorExempted)
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
		it.Event = new(Erc20BridgeLogicRestrictedDepositorExempted)
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
func (it *Erc20BridgeLogicRestrictedDepositorExemptedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedDepositorExemptedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedDepositorExempted represents a DepositorExempted event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedDepositorExempted struct {
	Depositor common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDepositorExempted is a free log retrieval operation binding the contract event 0xf56e0868b913034a60dbca9c89ee79f8b0fa18dadbc5f6665f2f9a2cf3f51cdb.
//
// Solidity: event Depositor_Exempted(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterDepositorExempted(opts *bind.FilterOpts, depositor []common.Address) (*Erc20BridgeLogicRestrictedDepositorExemptedIterator, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Depositor_Exempted", depositorRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedDepositorExemptedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Depositor_Exempted", logs: logs, sub: sub}, nil
}

// WatchDepositorExempted is a free log subscription operation binding the contract event 0xf56e0868b913034a60dbca9c89ee79f8b0fa18dadbc5f6665f2f9a2cf3f51cdb.
//
// Solidity: event Depositor_Exempted(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchDepositorExempted(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedDepositorExempted, depositor []common.Address) (event.Subscription, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Depositor_Exempted", depositorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedDepositorExempted)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Depositor_Exempted", log); err != nil {
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

// ParseDepositorExempted is a log parse operation binding the contract event 0xf56e0868b913034a60dbca9c89ee79f8b0fa18dadbc5f6665f2f9a2cf3f51cdb.
//
// Solidity: event Depositor_Exempted(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseDepositorExempted(log types.Log) (*Erc20BridgeLogicRestrictedDepositorExempted, error) {
	event := new(Erc20BridgeLogicRestrictedDepositorExempted)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Depositor_Exempted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator is returned from FilterDepositorExemptionRevoked and is used to iterate over the raw logs and unpacked data for DepositorExemptionRevoked events raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator struct {
	Event *Erc20BridgeLogicRestrictedDepositorExemptionRevoked // Event containing the contract specifics and raw log

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
func (it *Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20BridgeLogicRestrictedDepositorExemptionRevoked)
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
		it.Event = new(Erc20BridgeLogicRestrictedDepositorExemptionRevoked)
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
func (it *Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20BridgeLogicRestrictedDepositorExemptionRevoked represents a DepositorExemptionRevoked event raised by the Erc20BridgeLogicRestricted contract.
type Erc20BridgeLogicRestrictedDepositorExemptionRevoked struct {
	Depositor common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterDepositorExemptionRevoked is a free log retrieval operation binding the contract event 0xe74b113dca87276d976f476a9b4b9da3c780a3262eaabad051ee4e98912936a4.
//
// Solidity: event Depositor_Exemption_Revoked(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterDepositorExemptionRevoked(opts *bind.FilterOpts, depositor []common.Address) (*Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "Depositor_Exemption_Revoked", depositorRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "Depositor_Exemption_Revoked", logs: logs, sub: sub}, nil
}

// WatchDepositorExemptionRevoked is a free log subscription operation binding the contract event 0xe74b113dca87276d976f476a9b4b9da3c780a3262eaabad051ee4e98912936a4.
//
// Solidity: event Depositor_Exemption_Revoked(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchDepositorExemptionRevoked(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedDepositorExemptionRevoked, depositor []common.Address) (event.Subscription, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "Depositor_Exemption_Revoked", depositorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20BridgeLogicRestrictedDepositorExemptionRevoked)
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Depositor_Exemption_Revoked", log); err != nil {
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

// ParseDepositorExemptionRevoked is a log parse operation binding the contract event 0xe74b113dca87276d976f476a9b4b9da3c780a3262eaabad051ee4e98912936a4.
//
// Solidity: event Depositor_Exemption_Revoked(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseDepositorExemptionRevoked(log types.Log) (*Erc20BridgeLogicRestrictedDepositorExemptionRevoked, error) {
	event := new(Erc20BridgeLogicRestrictedDepositorExemptionRevoked)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "Depositor_Exemption_Revoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
