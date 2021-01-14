// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bridge

import (
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
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// BridgeABI is the input ABI used to generate the binding from.
const BridgeABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"erc20_asset_pool\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"multisig_control\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"new_minimum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Deposit_Minimum_Set\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user_address\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"Asset_Deposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Listed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Removed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user_address\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Withdrawn\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"list_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"remove_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"minimum_amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"set_deposit_minimum\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiry\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"withdraw_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"deposit_asset\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"is_asset_listed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"get_deposit_minimum\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"get_multisig_control_address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"}],\"name\":\"get_vega_id\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"}],\"name\":\"get_asset_source\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

// Bridge is an auto generated Go binding around an Ethereum contract.
type Bridge struct {
	BridgeCaller     // Read-only binding to the contract
	BridgeTransactor // Write-only binding to the contract
	BridgeFilterer   // Log filterer for contract events
}

// BridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeSession struct {
	Contract     *Bridge           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeCallerSession struct {
	Contract *BridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// BridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeTransactorSession struct {
	Contract     *BridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeRaw struct {
	Contract *Bridge // Generic contract binding to access the raw methods on
}

// BridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeCallerRaw struct {
	Contract *BridgeCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeTransactorRaw struct {
	Contract *BridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridge creates a new instance of Bridge, bound to a specific deployed contract.
func NewBridge(address common.Address, backend bind.ContractBackend) (*Bridge, error) {
	contract, err := bindBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Bridge{BridgeCaller: BridgeCaller{contract: contract}, BridgeTransactor: BridgeTransactor{contract: contract}, BridgeFilterer: BridgeFilterer{contract: contract}}, nil
}

// NewBridgeCaller creates a new read-only instance of Bridge, bound to a specific deployed contract.
func NewBridgeCaller(address common.Address, caller bind.ContractCaller) (*BridgeCaller, error) {
	contract, err := bindBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeCaller{contract: contract}, nil
}

// NewBridgeTransactor creates a new write-only instance of Bridge, bound to a specific deployed contract.
func NewBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeTransactor, error) {
	contract, err := bindBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeTransactor{contract: contract}, nil
}

// NewBridgeFilterer creates a new log filterer instance of Bridge, bound to a specific deployed contract.
func NewBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeFilterer, error) {
	contract, err := bindBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeFilterer{contract: contract}, nil
}

// bindBridge binds a generic wrapper to an already deployed contract.
func bindBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BridgeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bridge *BridgeRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Bridge.Contract.BridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bridge *BridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.Contract.BridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bridge *BridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bridge.Contract.BridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Bridge *BridgeCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Bridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Bridge *BridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Bridge *BridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Bridge.Contract.contract.Transact(opts, method, params...)
}

// GetAssetSource is a free data retrieval call binding the contract method 0x786b0bc0.
//
// Solidity: function get_asset_source(bytes32 vega_id) view returns(address)
func (_Bridge *BridgeCaller) GetAssetSource(opts *bind.CallOpts, vega_id [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "get_asset_source", vega_id)
	return *ret0, err
}

// GetAssetSource is a free data retrieval call binding the contract method 0x786b0bc0.
//
// Solidity: function get_asset_source(bytes32 vega_id) view returns(address)
func (_Bridge *BridgeSession) GetAssetSource(vega_id [32]byte) (common.Address, error) {
	return _Bridge.Contract.GetAssetSource(&_Bridge.CallOpts, vega_id)
}

// GetAssetSource is a free data retrieval call binding the contract method 0x786b0bc0.
//
// Solidity: function get_asset_source(bytes32 vega_id) view returns(address)
func (_Bridge *BridgeCallerSession) GetAssetSource(vega_id [32]byte) (common.Address, error) {
	return _Bridge.Contract.GetAssetSource(&_Bridge.CallOpts, vega_id)
}

// GetDepositMinimum is a free data retrieval call binding the contract method 0x4322b1f2.
//
// Solidity: function get_deposit_minimum(address asset_source) view returns(uint256)
func (_Bridge *BridgeCaller) GetDepositMinimum(opts *bind.CallOpts, asset_source common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "get_deposit_minimum", asset_source)
	return *ret0, err
}

// GetDepositMinimum is a free data retrieval call binding the contract method 0x4322b1f2.
//
// Solidity: function get_deposit_minimum(address asset_source) view returns(uint256)
func (_Bridge *BridgeSession) GetDepositMinimum(asset_source common.Address) (*big.Int, error) {
	return _Bridge.Contract.GetDepositMinimum(&_Bridge.CallOpts, asset_source)
}

// GetDepositMinimum is a free data retrieval call binding the contract method 0x4322b1f2.
//
// Solidity: function get_deposit_minimum(address asset_source) view returns(uint256)
func (_Bridge *BridgeCallerSession) GetDepositMinimum(asset_source common.Address) (*big.Int, error) {
	return _Bridge.Contract.GetDepositMinimum(&_Bridge.CallOpts, asset_source)
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0xc58dc3b9.
//
// Solidity: function get_multisig_control_address() view returns(address)
func (_Bridge *BridgeCaller) GetMultisigControlAddress(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "get_multisig_control_address")
	return *ret0, err
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0xc58dc3b9.
//
// Solidity: function get_multisig_control_address() view returns(address)
func (_Bridge *BridgeSession) GetMultisigControlAddress() (common.Address, error) {
	return _Bridge.Contract.GetMultisigControlAddress(&_Bridge.CallOpts)
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0xc58dc3b9.
//
// Solidity: function get_multisig_control_address() view returns(address)
func (_Bridge *BridgeCallerSession) GetMultisigControlAddress() (common.Address, error) {
	return _Bridge.Contract.GetMultisigControlAddress(&_Bridge.CallOpts)
}

// GetVegaId is a free data retrieval call binding the contract method 0x28ee726e.
//
// Solidity: function get_vega_id(address asset_source) view returns(bytes32)
func (_Bridge *BridgeCaller) GetVegaId(opts *bind.CallOpts, asset_source common.Address) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "get_vega_id", asset_source)
	return *ret0, err
}

// GetVegaId is a free data retrieval call binding the contract method 0x28ee726e.
//
// Solidity: function get_vega_id(address asset_source) view returns(bytes32)
func (_Bridge *BridgeSession) GetVegaId(asset_source common.Address) ([32]byte, error) {
	return _Bridge.Contract.GetVegaId(&_Bridge.CallOpts, asset_source)
}

// GetVegaId is a free data retrieval call binding the contract method 0x28ee726e.
//
// Solidity: function get_vega_id(address asset_source) view returns(bytes32)
func (_Bridge *BridgeCallerSession) GetVegaId(asset_source common.Address) ([32]byte, error) {
	return _Bridge.Contract.GetVegaId(&_Bridge.CallOpts, asset_source)
}

// IsAssetListed is a free data retrieval call binding the contract method 0x7fd27b7f.
//
// Solidity: function is_asset_listed(address asset_source) view returns(bool)
func (_Bridge *BridgeCaller) IsAssetListed(opts *bind.CallOpts, asset_source common.Address) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "is_asset_listed", asset_source)
	return *ret0, err
}

// IsAssetListed is a free data retrieval call binding the contract method 0x7fd27b7f.
//
// Solidity: function is_asset_listed(address asset_source) view returns(bool)
func (_Bridge *BridgeSession) IsAssetListed(asset_source common.Address) (bool, error) {
	return _Bridge.Contract.IsAssetListed(&_Bridge.CallOpts, asset_source)
}

// IsAssetListed is a free data retrieval call binding the contract method 0x7fd27b7f.
//
// Solidity: function is_asset_listed(address asset_source) view returns(bool)
func (_Bridge *BridgeCallerSession) IsAssetListed(asset_source common.Address) (bool, error) {
	return _Bridge.Contract.IsAssetListed(&_Bridge.CallOpts, asset_source)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf7683932.
//
// Solidity: function deposit_asset(address asset_source, uint256 amount, bytes32 vega_public_key) returns()
func (_Bridge *BridgeTransactor) DepositAsset(opts *bind.TransactOpts, asset_source common.Address, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "deposit_asset", asset_source, amount, vega_public_key)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf7683932.
//
// Solidity: function deposit_asset(address asset_source, uint256 amount, bytes32 vega_public_key) returns()
func (_Bridge *BridgeSession) DepositAsset(asset_source common.Address, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.DepositAsset(&_Bridge.TransactOpts, asset_source, amount, vega_public_key)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf7683932.
//
// Solidity: function deposit_asset(address asset_source, uint256 amount, bytes32 vega_public_key) returns()
func (_Bridge *BridgeTransactorSession) DepositAsset(asset_source common.Address, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.DepositAsset(&_Bridge.TransactOpts, asset_source, amount, vega_public_key)
}

// ListAsset is a paid mutator transaction binding the contract method 0xa8780cda.
//
// Solidity: function list_asset(address asset_source, bytes32 vega_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) ListAsset(opts *bind.TransactOpts, asset_source common.Address, vega_id [32]byte, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "list_asset", asset_source, vega_id, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0xa8780cda.
//
// Solidity: function list_asset(address asset_source, bytes32 vega_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) ListAsset(asset_source common.Address, vega_id [32]byte, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.ListAsset(&_Bridge.TransactOpts, asset_source, vega_id, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0xa8780cda.
//
// Solidity: function list_asset(address asset_source, bytes32 vega_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) ListAsset(asset_source common.Address, vega_id [32]byte, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.ListAsset(&_Bridge.TransactOpts, asset_source, vega_id, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc76de358.
//
// Solidity: function remove_asset(address asset_source, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) RemoveAsset(opts *bind.TransactOpts, asset_source common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "remove_asset", asset_source, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc76de358.
//
// Solidity: function remove_asset(address asset_source, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) RemoveAsset(asset_source common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.RemoveAsset(&_Bridge.TransactOpts, asset_source, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc76de358.
//
// Solidity: function remove_asset(address asset_source, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) RemoveAsset(asset_source common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.RemoveAsset(&_Bridge.TransactOpts, asset_source, nonce, signatures)
}

// SetDepositMinimum is a paid mutator transaction binding the contract method 0x3882b3da.
//
// Solidity: function set_deposit_minimum(address asset_source, uint256 minimum_amount, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) SetDepositMinimum(opts *bind.TransactOpts, asset_source common.Address, minimum_amount *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "set_deposit_minimum", asset_source, minimum_amount, nonce, signatures)
}

// SetDepositMinimum is a paid mutator transaction binding the contract method 0x3882b3da.
//
// Solidity: function set_deposit_minimum(address asset_source, uint256 minimum_amount, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) SetDepositMinimum(asset_source common.Address, minimum_amount *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.SetDepositMinimum(&_Bridge.TransactOpts, asset_source, minimum_amount, nonce, signatures)
}

// SetDepositMinimum is a paid mutator transaction binding the contract method 0x3882b3da.
//
// Solidity: function set_deposit_minimum(address asset_source, uint256 minimum_amount, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) SetDepositMinimum(asset_source common.Address, minimum_amount *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.SetDepositMinimum(&_Bridge.TransactOpts, asset_source, minimum_amount, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x7776a2a5.
//
// Solidity: function withdraw_asset(address asset_source, uint256 amount, uint256 expiry, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) WithdrawAsset(opts *bind.TransactOpts, asset_source common.Address, amount *big.Int, expiry *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "withdraw_asset", asset_source, amount, expiry, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x7776a2a5.
//
// Solidity: function withdraw_asset(address asset_source, uint256 amount, uint256 expiry, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) WithdrawAsset(asset_source common.Address, amount *big.Int, expiry *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.WithdrawAsset(&_Bridge.TransactOpts, asset_source, amount, expiry, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x7776a2a5.
//
// Solidity: function withdraw_asset(address asset_source, uint256 amount, uint256 expiry, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) WithdrawAsset(asset_source common.Address, amount *big.Int, expiry *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.WithdrawAsset(&_Bridge.TransactOpts, asset_source, amount, expiry, nonce, signatures)
}

// BridgeAssetDepositMinimumSetIterator is returned from FilterAssetDepositMinimumSet and is used to iterate over the raw logs and unpacked data for AssetDepositMinimumSet events raised by the Bridge contract.
type BridgeAssetDepositMinimumSetIterator struct {
	Event *BridgeAssetDepositMinimumSet // Event containing the contract specifics and raw log

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
func (it *BridgeAssetDepositMinimumSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeAssetDepositMinimumSet)
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
		it.Event = new(BridgeAssetDepositMinimumSet)
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
func (it *BridgeAssetDepositMinimumSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeAssetDepositMinimumSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeAssetDepositMinimumSet represents a AssetDepositMinimumSet event raised by the Bridge contract.
type BridgeAssetDepositMinimumSet struct {
	AssetSource common.Address
	NewMinimum  *big.Int
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetDepositMinimumSet is a free log retrieval operation binding the contract event 0x4ed0df0b169b573722ecdcc12333646a6efd0445c28fe277470bbfca620e8ad5.
//
// Solidity: event Asset_Deposit_Minimum_Set(address indexed asset_source, uint256 new_minimum, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetDepositMinimumSet(opts *bind.FilterOpts, asset_source []common.Address) (*BridgeAssetDepositMinimumSetIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Deposit_Minimum_Set", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetDepositMinimumSetIterator{contract: _Bridge.contract, event: "Asset_Deposit_Minimum_Set", logs: logs, sub: sub}, nil
}

// WatchAssetDepositMinimumSet is a free log subscription operation binding the contract event 0x4ed0df0b169b573722ecdcc12333646a6efd0445c28fe277470bbfca620e8ad5.
//
// Solidity: event Asset_Deposit_Minimum_Set(address indexed asset_source, uint256 new_minimum, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetDepositMinimumSet(opts *bind.WatchOpts, sink chan<- *BridgeAssetDepositMinimumSet, asset_source []common.Address) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Deposit_Minimum_Set", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeAssetDepositMinimumSet)
				if err := _Bridge.contract.UnpackLog(event, "Asset_Deposit_Minimum_Set", log); err != nil {
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

// ParseAssetDepositMinimumSet is a log parse operation binding the contract event 0x4ed0df0b169b573722ecdcc12333646a6efd0445c28fe277470bbfca620e8ad5.
//
// Solidity: event Asset_Deposit_Minimum_Set(address indexed asset_source, uint256 new_minimum, uint256 nonce)
func (_Bridge *BridgeFilterer) ParseAssetDepositMinimumSet(log types.Log) (*BridgeAssetDepositMinimumSet, error) {
	event := new(BridgeAssetDepositMinimumSet)
	if err := _Bridge.contract.UnpackLog(event, "Asset_Deposit_Minimum_Set", log); err != nil {
		return nil, err
	}
	return event, nil
}

// BridgeAssetDepositedIterator is returned from FilterAssetDeposited and is used to iterate over the raw logs and unpacked data for AssetDeposited events raised by the Bridge contract.
type BridgeAssetDepositedIterator struct {
	Event *BridgeAssetDeposited // Event containing the contract specifics and raw log

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
func (it *BridgeAssetDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeAssetDeposited)
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
		it.Event = new(BridgeAssetDeposited)
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
func (it *BridgeAssetDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeAssetDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeAssetDeposited represents a AssetDeposited event raised by the Bridge contract.
type BridgeAssetDeposited struct {
	UserAddress   common.Address
	AssetSource   common.Address
	Amount        *big.Int
	VegaPublicKey [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAssetDeposited is a free log retrieval operation binding the contract event 0x3724ff5e82ddc640a08d68b0b782a5991aea0de51a8dd10a59cdbe5b3ec4e6bf.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 amount, bytes32 vega_public_key)
func (_Bridge *BridgeFilterer) FilterAssetDeposited(opts *bind.FilterOpts, user_address []common.Address, asset_source []common.Address) (*BridgeAssetDepositedIterator, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Deposited", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetDepositedIterator{contract: _Bridge.contract, event: "Asset_Deposited", logs: logs, sub: sub}, nil
}

// WatchAssetDeposited is a free log subscription operation binding the contract event 0x3724ff5e82ddc640a08d68b0b782a5991aea0de51a8dd10a59cdbe5b3ec4e6bf.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 amount, bytes32 vega_public_key)
func (_Bridge *BridgeFilterer) WatchAssetDeposited(opts *bind.WatchOpts, sink chan<- *BridgeAssetDeposited, user_address []common.Address, asset_source []common.Address) (event.Subscription, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Deposited", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeAssetDeposited)
				if err := _Bridge.contract.UnpackLog(event, "Asset_Deposited", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseAssetDeposited(log types.Log) (*BridgeAssetDeposited, error) {
	event := new(BridgeAssetDeposited)
	if err := _Bridge.contract.UnpackLog(event, "Asset_Deposited", log); err != nil {
		return nil, err
	}
	return event, nil
}

// BridgeAssetListedIterator is returned from FilterAssetListed and is used to iterate over the raw logs and unpacked data for AssetListed events raised by the Bridge contract.
type BridgeAssetListedIterator struct {
	Event *BridgeAssetListed // Event containing the contract specifics and raw log

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
func (it *BridgeAssetListedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeAssetListed)
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
		it.Event = new(BridgeAssetListed)
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
func (it *BridgeAssetListedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeAssetListedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeAssetListed represents a AssetListed event raised by the Bridge contract.
type BridgeAssetListed struct {
	AssetSource common.Address
	VegaId      [32]byte
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetListed is a free log retrieval operation binding the contract event 0x4180d77d05ff0d31650c548c23f2de07a3da3ad42e3dd6edd817b438a150452e.
//
// Solidity: event Asset_Listed(address indexed asset_source, bytes32 indexed vega_id, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetListed(opts *bind.FilterOpts, asset_source []common.Address, vega_id [][32]byte) (*BridgeAssetListedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var vega_idRule []interface{}
	for _, vega_idItem := range vega_id {
		vega_idRule = append(vega_idRule, vega_idItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Listed", asset_sourceRule, vega_idRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetListedIterator{contract: _Bridge.contract, event: "Asset_Listed", logs: logs, sub: sub}, nil
}

// WatchAssetListed is a free log subscription operation binding the contract event 0x4180d77d05ff0d31650c548c23f2de07a3da3ad42e3dd6edd817b438a150452e.
//
// Solidity: event Asset_Listed(address indexed asset_source, bytes32 indexed vega_id, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetListed(opts *bind.WatchOpts, sink chan<- *BridgeAssetListed, asset_source []common.Address, vega_id [][32]byte) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var vega_idRule []interface{}
	for _, vega_idItem := range vega_id {
		vega_idRule = append(vega_idRule, vega_idItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Listed", asset_sourceRule, vega_idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeAssetListed)
				if err := _Bridge.contract.UnpackLog(event, "Asset_Listed", log); err != nil {
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
// Solidity: event Asset_Listed(address indexed asset_source, bytes32 indexed vega_id, uint256 nonce)
func (_Bridge *BridgeFilterer) ParseAssetListed(log types.Log) (*BridgeAssetListed, error) {
	event := new(BridgeAssetListed)
	if err := _Bridge.contract.UnpackLog(event, "Asset_Listed", log); err != nil {
		return nil, err
	}
	return event, nil
}

// BridgeAssetRemovedIterator is returned from FilterAssetRemoved and is used to iterate over the raw logs and unpacked data for AssetRemoved events raised by the Bridge contract.
type BridgeAssetRemovedIterator struct {
	Event *BridgeAssetRemoved // Event containing the contract specifics and raw log

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
func (it *BridgeAssetRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeAssetRemoved)
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
		it.Event = new(BridgeAssetRemoved)
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
func (it *BridgeAssetRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeAssetRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeAssetRemoved represents a AssetRemoved event raised by the Bridge contract.
type BridgeAssetRemoved struct {
	AssetSource common.Address
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetRemoved is a free log retrieval operation binding the contract event 0x58ad5e799e2df93ab408be0e5c1870d44c80b5bca99dfaf7ddf0dab5e6b155c9.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetRemoved(opts *bind.FilterOpts, asset_source []common.Address) (*BridgeAssetRemovedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Removed", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetRemovedIterator{contract: _Bridge.contract, event: "Asset_Removed", logs: logs, sub: sub}, nil
}

// WatchAssetRemoved is a free log subscription operation binding the contract event 0x58ad5e799e2df93ab408be0e5c1870d44c80b5bca99dfaf7ddf0dab5e6b155c9.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetRemoved(opts *bind.WatchOpts, sink chan<- *BridgeAssetRemoved, asset_source []common.Address) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Removed", asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeAssetRemoved)
				if err := _Bridge.contract.UnpackLog(event, "Asset_Removed", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseAssetRemoved(log types.Log) (*BridgeAssetRemoved, error) {
	event := new(BridgeAssetRemoved)
	if err := _Bridge.contract.UnpackLog(event, "Asset_Removed", log); err != nil {
		return nil, err
	}
	return event, nil
}

// BridgeAssetWithdrawnIterator is returned from FilterAssetWithdrawn and is used to iterate over the raw logs and unpacked data for AssetWithdrawn events raised by the Bridge contract.
type BridgeAssetWithdrawnIterator struct {
	Event *BridgeAssetWithdrawn // Event containing the contract specifics and raw log

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
func (it *BridgeAssetWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeAssetWithdrawn)
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
		it.Event = new(BridgeAssetWithdrawn)
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
func (it *BridgeAssetWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeAssetWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeAssetWithdrawn represents a AssetWithdrawn event raised by the Bridge contract.
type BridgeAssetWithdrawn struct {
	UserAddress common.Address
	AssetSource common.Address
	Amount      *big.Int
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetWithdrawn is a free log retrieval operation binding the contract event 0xa79be4f3361e32d396d64c478ecef73732cb40b2a75702c3b3b3226a2c83b5df.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 amount, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetWithdrawn(opts *bind.FilterOpts, user_address []common.Address, asset_source []common.Address) (*BridgeAssetWithdrawnIterator, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Withdrawn", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetWithdrawnIterator{contract: _Bridge.contract, event: "Asset_Withdrawn", logs: logs, sub: sub}, nil
}

// WatchAssetWithdrawn is a free log subscription operation binding the contract event 0xa79be4f3361e32d396d64c478ecef73732cb40b2a75702c3b3b3226a2c83b5df.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 amount, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetWithdrawn(opts *bind.WatchOpts, sink chan<- *BridgeAssetWithdrawn, user_address []common.Address, asset_source []common.Address) (event.Subscription, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Withdrawn", user_addressRule, asset_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeAssetWithdrawn)
				if err := _Bridge.contract.UnpackLog(event, "Asset_Withdrawn", log); err != nil {
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
func (_Bridge *BridgeFilterer) ParseAssetWithdrawn(log types.Log) (*BridgeAssetWithdrawn, error) {
	event := new(BridgeAssetWithdrawn)
	if err := _Bridge.contract.UnpackLog(event, "Asset_Withdrawn", log); err != nil {
		return nil, err
	}
	return event, nil
}
