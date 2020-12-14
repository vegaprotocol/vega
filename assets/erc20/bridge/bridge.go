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
const BridgeABI = "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"erc20_asset_pool\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"multisig_control\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"new_minimum\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Deposit_Minimum_Set\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user_address\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"Asset_Deposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Listed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Removed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"user_address\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"}],\"name\":\"Asset_Withdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"multisig_control_source\",\"type\":\"address\"}],\"name\":\"Multisig_Control_Set\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"constant\":true,\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"kill\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"}],\"name\":\"list_asset_admin\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"}],\"name\":\"remove_asset_admin\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"}],\"name\":\"get_vega_id\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"}],\"name\":\"get_asset_source_and_asset_id\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"vega_id\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"list_asset\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"remove_asset\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"minimum_amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"set_deposit_minimum\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiry\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signatures\",\"type\":\"bytes\"}],\"name\":\"withdraw_asset\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"vega_public_key\",\"type\":\"bytes32\"}],\"name\":\"deposit_asset\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"}],\"name\":\"is_asset_listed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"internalType\":\"address\",\"name\":\"asset_source\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"asset_id\",\"type\":\"uint256\"}],\"name\":\"get_deposit_minimum\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"get_multisig_control_address\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

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

// GetAssetSourceAndAssetId is a free data retrieval call binding the contract method 0xcc1e14ef.
//
// Solidity: function get_asset_source_and_asset_id(bytes32 vega_id) view returns(address, uint256)
func (_Bridge *BridgeCaller) GetAssetSourceAndAssetId(opts *bind.CallOpts, vega_id [32]byte) (common.Address, *big.Int, error) {
	var (
		ret0 = new(common.Address)
		ret1 = new(*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _Bridge.contract.Call(opts, out, "get_asset_source_and_asset_id", vega_id)
	return *ret0, *ret1, err
}

// GetAssetSourceAndAssetId is a free data retrieval call binding the contract method 0xcc1e14ef.
//
// Solidity: function get_asset_source_and_asset_id(bytes32 vega_id) view returns(address, uint256)
func (_Bridge *BridgeSession) GetAssetSourceAndAssetId(vega_id [32]byte) (common.Address, *big.Int, error) {
	return _Bridge.Contract.GetAssetSourceAndAssetId(&_Bridge.CallOpts, vega_id)
}

// GetAssetSourceAndAssetId is a free data retrieval call binding the contract method 0xcc1e14ef.
//
// Solidity: function get_asset_source_and_asset_id(bytes32 vega_id) view returns(address, uint256)
func (_Bridge *BridgeCallerSession) GetAssetSourceAndAssetId(vega_id [32]byte) (common.Address, *big.Int, error) {
	return _Bridge.Contract.GetAssetSourceAndAssetId(&_Bridge.CallOpts, vega_id)
}

// GetDepositMinimum is a free data retrieval call binding the contract method 0xa9d9e9f0.
//
// Solidity: function get_deposit_minimum(address asset_source, uint256 asset_id) view returns(uint256)
func (_Bridge *BridgeCaller) GetDepositMinimum(opts *bind.CallOpts, asset_source common.Address, asset_id *big.Int) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "get_deposit_minimum", asset_source, asset_id)
	return *ret0, err
}

// GetDepositMinimum is a free data retrieval call binding the contract method 0xa9d9e9f0.
//
// Solidity: function get_deposit_minimum(address asset_source, uint256 asset_id) view returns(uint256)
func (_Bridge *BridgeSession) GetDepositMinimum(asset_source common.Address, asset_id *big.Int) (*big.Int, error) {
	return _Bridge.Contract.GetDepositMinimum(&_Bridge.CallOpts, asset_source, asset_id)
}

// GetDepositMinimum is a free data retrieval call binding the contract method 0xa9d9e9f0.
//
// Solidity: function get_deposit_minimum(address asset_source, uint256 asset_id) view returns(uint256)
func (_Bridge *BridgeCallerSession) GetDepositMinimum(asset_source common.Address, asset_id *big.Int) (*big.Int, error) {
	return _Bridge.Contract.GetDepositMinimum(&_Bridge.CallOpts, asset_source, asset_id)
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

// GetVegaId is a free data retrieval call binding the contract method 0xe9da406e.
//
// Solidity: function get_vega_id(address asset_source, uint256 asset_id) view returns(bytes32)
func (_Bridge *BridgeCaller) GetVegaId(opts *bind.CallOpts, asset_source common.Address, asset_id *big.Int) ([32]byte, error) {
	var (
		ret0 = new([32]byte)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "get_vega_id", asset_source, asset_id)
	return *ret0, err
}

// GetVegaId is a free data retrieval call binding the contract method 0xe9da406e.
//
// Solidity: function get_vega_id(address asset_source, uint256 asset_id) view returns(bytes32)
func (_Bridge *BridgeSession) GetVegaId(asset_source common.Address, asset_id *big.Int) ([32]byte, error) {
	return _Bridge.Contract.GetVegaId(&_Bridge.CallOpts, asset_source, asset_id)
}

// GetVegaId is a free data retrieval call binding the contract method 0xe9da406e.
//
// Solidity: function get_vega_id(address asset_source, uint256 asset_id) view returns(bytes32)
func (_Bridge *BridgeCallerSession) GetVegaId(asset_source common.Address, asset_id *big.Int) ([32]byte, error) {
	return _Bridge.Contract.GetVegaId(&_Bridge.CallOpts, asset_source, asset_id)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Bridge *BridgeCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "isOwner")
	return *ret0, err
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Bridge *BridgeSession) IsOwner() (bool, error) {
	return _Bridge.Contract.IsOwner(&_Bridge.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Bridge *BridgeCallerSession) IsOwner() (bool, error) {
	return _Bridge.Contract.IsOwner(&_Bridge.CallOpts)
}

// IsAssetListed is a free data retrieval call binding the contract method 0xb52d5507.
//
// Solidity: function is_asset_listed(address asset_source, uint256 asset_id) view returns(bool)
func (_Bridge *BridgeCaller) IsAssetListed(opts *bind.CallOpts, asset_source common.Address, asset_id *big.Int) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "is_asset_listed", asset_source, asset_id)
	return *ret0, err
}

// IsAssetListed is a free data retrieval call binding the contract method 0xb52d5507.
//
// Solidity: function is_asset_listed(address asset_source, uint256 asset_id) view returns(bool)
func (_Bridge *BridgeSession) IsAssetListed(asset_source common.Address, asset_id *big.Int) (bool, error) {
	return _Bridge.Contract.IsAssetListed(&_Bridge.CallOpts, asset_source, asset_id)
}

// IsAssetListed is a free data retrieval call binding the contract method 0xb52d5507.
//
// Solidity: function is_asset_listed(address asset_source, uint256 asset_id) view returns(bool)
func (_Bridge *BridgeCallerSession) IsAssetListed(asset_source common.Address, asset_id *big.Int) (bool, error) {
	return _Bridge.Contract.IsAssetListed(&_Bridge.CallOpts, asset_source, asset_id)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Bridge *BridgeCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Bridge.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Bridge *BridgeSession) Owner() (common.Address, error) {
	return _Bridge.Contract.Owner(&_Bridge.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Bridge *BridgeCallerSession) Owner() (common.Address, error) {
	return _Bridge.Contract.Owner(&_Bridge.CallOpts)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf9222544.
//
// Solidity: function deposit_asset(address asset_source, uint256 asset_id, uint256 amount, bytes32 vega_public_key) returns()
func (_Bridge *BridgeTransactor) DepositAsset(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "deposit_asset", asset_source, asset_id, amount, vega_public_key)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf9222544.
//
// Solidity: function deposit_asset(address asset_source, uint256 asset_id, uint256 amount, bytes32 vega_public_key) returns()
func (_Bridge *BridgeSession) DepositAsset(asset_source common.Address, asset_id *big.Int, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.DepositAsset(&_Bridge.TransactOpts, asset_source, asset_id, amount, vega_public_key)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xf9222544.
//
// Solidity: function deposit_asset(address asset_source, uint256 asset_id, uint256 amount, bytes32 vega_public_key) returns()
func (_Bridge *BridgeTransactorSession) DepositAsset(asset_source common.Address, asset_id *big.Int, amount *big.Int, vega_public_key [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.DepositAsset(&_Bridge.TransactOpts, asset_source, asset_id, amount, vega_public_key)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Bridge *BridgeTransactor) Kill(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "kill")
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Bridge *BridgeSession) Kill() (*types.Transaction, error) {
	return _Bridge.Contract.Kill(&_Bridge.TransactOpts)
}

// Kill is a paid mutator transaction binding the contract method 0x41c0e1b5.
//
// Solidity: function kill() returns()
func (_Bridge *BridgeTransactorSession) Kill() (*types.Transaction, error) {
	return _Bridge.Contract.Kill(&_Bridge.TransactOpts)
}

// ListAsset is a paid mutator transaction binding the contract method 0x4e818110.
//
// Solidity: function list_asset(address asset_source, uint256 asset_id, bytes32 vega_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) ListAsset(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int, vega_id [32]byte, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "list_asset", asset_source, asset_id, vega_id, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x4e818110.
//
// Solidity: function list_asset(address asset_source, uint256 asset_id, bytes32 vega_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) ListAsset(asset_source common.Address, asset_id *big.Int, vega_id [32]byte, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.ListAsset(&_Bridge.TransactOpts, asset_source, asset_id, vega_id, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x4e818110.
//
// Solidity: function list_asset(address asset_source, uint256 asset_id, bytes32 vega_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) ListAsset(asset_source common.Address, asset_id *big.Int, vega_id [32]byte, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.ListAsset(&_Bridge.TransactOpts, asset_source, asset_id, vega_id, nonce, signatures)
}

// ListAssetAdmin is a paid mutator transaction binding the contract method 0x6ceb085a.
//
// Solidity: function list_asset_admin(address asset_source, uint256 asset_id, bytes32 vega_id) returns()
func (_Bridge *BridgeTransactor) ListAssetAdmin(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int, vega_id [32]byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "list_asset_admin", asset_source, asset_id, vega_id)
}

// ListAssetAdmin is a paid mutator transaction binding the contract method 0x6ceb085a.
//
// Solidity: function list_asset_admin(address asset_source, uint256 asset_id, bytes32 vega_id) returns()
func (_Bridge *BridgeSession) ListAssetAdmin(asset_source common.Address, asset_id *big.Int, vega_id [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.ListAssetAdmin(&_Bridge.TransactOpts, asset_source, asset_id, vega_id)
}

// ListAssetAdmin is a paid mutator transaction binding the contract method 0x6ceb085a.
//
// Solidity: function list_asset_admin(address asset_source, uint256 asset_id, bytes32 vega_id) returns()
func (_Bridge *BridgeTransactorSession) ListAssetAdmin(asset_source common.Address, asset_id *big.Int, vega_id [32]byte) (*types.Transaction, error) {
	return _Bridge.Contract.ListAssetAdmin(&_Bridge.TransactOpts, asset_source, asset_id, vega_id)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc4cc885e.
//
// Solidity: function remove_asset(address asset_source, uint256 asset_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) RemoveAsset(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "remove_asset", asset_source, asset_id, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc4cc885e.
//
// Solidity: function remove_asset(address asset_source, uint256 asset_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) RemoveAsset(asset_source common.Address, asset_id *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.RemoveAsset(&_Bridge.TransactOpts, asset_source, asset_id, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0xc4cc885e.
//
// Solidity: function remove_asset(address asset_source, uint256 asset_id, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) RemoveAsset(asset_source common.Address, asset_id *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.RemoveAsset(&_Bridge.TransactOpts, asset_source, asset_id, nonce, signatures)
}

// RemoveAssetAdmin is a paid mutator transaction binding the contract method 0x2f735b36.
//
// Solidity: function remove_asset_admin(address asset_source, uint256 asset_id) returns()
func (_Bridge *BridgeTransactor) RemoveAssetAdmin(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "remove_asset_admin", asset_source, asset_id)
}

// RemoveAssetAdmin is a paid mutator transaction binding the contract method 0x2f735b36.
//
// Solidity: function remove_asset_admin(address asset_source, uint256 asset_id) returns()
func (_Bridge *BridgeSession) RemoveAssetAdmin(asset_source common.Address, asset_id *big.Int) (*types.Transaction, error) {
	return _Bridge.Contract.RemoveAssetAdmin(&_Bridge.TransactOpts, asset_source, asset_id)
}

// RemoveAssetAdmin is a paid mutator transaction binding the contract method 0x2f735b36.
//
// Solidity: function remove_asset_admin(address asset_source, uint256 asset_id) returns()
func (_Bridge *BridgeTransactorSession) RemoveAssetAdmin(asset_source common.Address, asset_id *big.Int) (*types.Transaction, error) {
	return _Bridge.Contract.RemoveAssetAdmin(&_Bridge.TransactOpts, asset_source, asset_id)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Bridge *BridgeTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Bridge *BridgeSession) RenounceOwnership() (*types.Transaction, error) {
	return _Bridge.Contract.RenounceOwnership(&_Bridge.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Bridge *BridgeTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Bridge.Contract.RenounceOwnership(&_Bridge.TransactOpts)
}

// SetDepositMinimum is a paid mutator transaction binding the contract method 0x98d6a1c1.
//
// Solidity: function set_deposit_minimum(address asset_source, uint256 asset_id, uint256 nonce, uint256 minimum_amount, bytes signatures) returns()
func (_Bridge *BridgeTransactor) SetDepositMinimum(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int, nonce *big.Int, minimum_amount *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "set_deposit_minimum", asset_source, asset_id, nonce, minimum_amount, signatures)
}

// SetDepositMinimum is a paid mutator transaction binding the contract method 0x98d6a1c1.
//
// Solidity: function set_deposit_minimum(address asset_source, uint256 asset_id, uint256 nonce, uint256 minimum_amount, bytes signatures) returns()
func (_Bridge *BridgeSession) SetDepositMinimum(asset_source common.Address, asset_id *big.Int, nonce *big.Int, minimum_amount *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.SetDepositMinimum(&_Bridge.TransactOpts, asset_source, asset_id, nonce, minimum_amount, signatures)
}

// SetDepositMinimum is a paid mutator transaction binding the contract method 0x98d6a1c1.
//
// Solidity: function set_deposit_minimum(address asset_source, uint256 asset_id, uint256 nonce, uint256 minimum_amount, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) SetDepositMinimum(asset_source common.Address, asset_id *big.Int, nonce *big.Int, minimum_amount *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.SetDepositMinimum(&_Bridge.TransactOpts, asset_source, asset_id, nonce, minimum_amount, signatures)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Bridge *BridgeTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Bridge *BridgeSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.TransferOwnership(&_Bridge.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Bridge *BridgeTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Bridge.Contract.TransferOwnership(&_Bridge.TransactOpts, newOwner)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x5287b256.
//
// Solidity: function withdraw_asset(address asset_source, uint256 asset_id, uint256 amount, uint256 expiry, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactor) WithdrawAsset(opts *bind.TransactOpts, asset_source common.Address, asset_id *big.Int, amount *big.Int, expiry *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.contract.Transact(opts, "withdraw_asset", asset_source, asset_id, amount, expiry, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x5287b256.
//
// Solidity: function withdraw_asset(address asset_source, uint256 asset_id, uint256 amount, uint256 expiry, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeSession) WithdrawAsset(asset_source common.Address, asset_id *big.Int, amount *big.Int, expiry *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.WithdrawAsset(&_Bridge.TransactOpts, asset_source, asset_id, amount, expiry, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x5287b256.
//
// Solidity: function withdraw_asset(address asset_source, uint256 asset_id, uint256 amount, uint256 expiry, uint256 nonce, bytes signatures) returns()
func (_Bridge *BridgeTransactorSession) WithdrawAsset(asset_source common.Address, asset_id *big.Int, amount *big.Int, expiry *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Bridge.Contract.WithdrawAsset(&_Bridge.TransactOpts, asset_source, asset_id, amount, expiry, nonce, signatures)
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
	AssetId     *big.Int
	NewMinimum  *big.Int
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetDepositMinimumSet is a free log retrieval operation binding the contract event 0x8bd6a2128f72cade235f8636548820a7baaf60321dded04c5222e1856e4900fd.
//
// Solidity: event Asset_Deposit_Minimum_Set(address indexed asset_source, uint256 indexed asset_id, uint256 new_minimum, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetDepositMinimumSet(opts *bind.FilterOpts, asset_source []common.Address, asset_id []*big.Int) (*BridgeAssetDepositMinimumSetIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Deposit_Minimum_Set", asset_sourceRule, asset_idRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetDepositMinimumSetIterator{contract: _Bridge.contract, event: "Asset_Deposit_Minimum_Set", logs: logs, sub: sub}, nil
}

// WatchAssetDepositMinimumSet is a free log subscription operation binding the contract event 0x8bd6a2128f72cade235f8636548820a7baaf60321dded04c5222e1856e4900fd.
//
// Solidity: event Asset_Deposit_Minimum_Set(address indexed asset_source, uint256 indexed asset_id, uint256 new_minimum, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetDepositMinimumSet(opts *bind.WatchOpts, sink chan<- *BridgeAssetDepositMinimumSet, asset_source []common.Address, asset_id []*big.Int) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Deposit_Minimum_Set", asset_sourceRule, asset_idRule)
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

// ParseAssetDepositMinimumSet is a log parse operation binding the contract event 0x8bd6a2128f72cade235f8636548820a7baaf60321dded04c5222e1856e4900fd.
//
// Solidity: event Asset_Deposit_Minimum_Set(address indexed asset_source, uint256 indexed asset_id, uint256 new_minimum, uint256 nonce)
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
	AssetId       *big.Int
	Amount        *big.Int
	VegaPublicKey [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterAssetDeposited is a free log retrieval operation binding the contract event 0x301354586185da7c1b383fec6b86a4f2afcd693fee1d3a4b60de799db0715dce.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 indexed asset_id, uint256 amount, bytes32 vega_public_key)
func (_Bridge *BridgeFilterer) FilterAssetDeposited(opts *bind.FilterOpts, user_address []common.Address, asset_source []common.Address, asset_id []*big.Int) (*BridgeAssetDepositedIterator, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Deposited", user_addressRule, asset_sourceRule, asset_idRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetDepositedIterator{contract: _Bridge.contract, event: "Asset_Deposited", logs: logs, sub: sub}, nil
}

// WatchAssetDeposited is a free log subscription operation binding the contract event 0x301354586185da7c1b383fec6b86a4f2afcd693fee1d3a4b60de799db0715dce.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 indexed asset_id, uint256 amount, bytes32 vega_public_key)
func (_Bridge *BridgeFilterer) WatchAssetDeposited(opts *bind.WatchOpts, sink chan<- *BridgeAssetDeposited, user_address []common.Address, asset_source []common.Address, asset_id []*big.Int) (event.Subscription, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Deposited", user_addressRule, asset_sourceRule, asset_idRule)
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

// ParseAssetDeposited is a log parse operation binding the contract event 0x301354586185da7c1b383fec6b86a4f2afcd693fee1d3a4b60de799db0715dce.
//
// Solidity: event Asset_Deposited(address indexed user_address, address indexed asset_source, uint256 indexed asset_id, uint256 amount, bytes32 vega_public_key)
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
	AssetId     *big.Int
	VegaId      [32]byte
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetListed is a free log retrieval operation binding the contract event 0xbe305435de3bbb45cc8cef562d95d86f1f4ffc1a39bae3f23721a9da33f073d7.
//
// Solidity: event Asset_Listed(address indexed asset_source, uint256 indexed asset_id, bytes32 indexed vega_id, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetListed(opts *bind.FilterOpts, asset_source []common.Address, asset_id []*big.Int, vega_id [][32]byte) (*BridgeAssetListedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}
	var vega_idRule []interface{}
	for _, vega_idItem := range vega_id {
		vega_idRule = append(vega_idRule, vega_idItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Listed", asset_sourceRule, asset_idRule, vega_idRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetListedIterator{contract: _Bridge.contract, event: "Asset_Listed", logs: logs, sub: sub}, nil
}

// WatchAssetListed is a free log subscription operation binding the contract event 0xbe305435de3bbb45cc8cef562d95d86f1f4ffc1a39bae3f23721a9da33f073d7.
//
// Solidity: event Asset_Listed(address indexed asset_source, uint256 indexed asset_id, bytes32 indexed vega_id, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetListed(opts *bind.WatchOpts, sink chan<- *BridgeAssetListed, asset_source []common.Address, asset_id []*big.Int, vega_id [][32]byte) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}
	var vega_idRule []interface{}
	for _, vega_idItem := range vega_id {
		vega_idRule = append(vega_idRule, vega_idItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Listed", asset_sourceRule, asset_idRule, vega_idRule)
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

// ParseAssetListed is a log parse operation binding the contract event 0xbe305435de3bbb45cc8cef562d95d86f1f4ffc1a39bae3f23721a9da33f073d7.
//
// Solidity: event Asset_Listed(address indexed asset_source, uint256 indexed asset_id, bytes32 indexed vega_id, uint256 nonce)
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
	AssetId     *big.Int
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetRemoved is a free log retrieval operation binding the contract event 0x8b7f47a4e62d4fffe89da8d5381345e897518068150bd10f0903d0a722c15774.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 indexed asset_id, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetRemoved(opts *bind.FilterOpts, asset_source []common.Address, asset_id []*big.Int) (*BridgeAssetRemovedIterator, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Removed", asset_sourceRule, asset_idRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetRemovedIterator{contract: _Bridge.contract, event: "Asset_Removed", logs: logs, sub: sub}, nil
}

// WatchAssetRemoved is a free log subscription operation binding the contract event 0x8b7f47a4e62d4fffe89da8d5381345e897518068150bd10f0903d0a722c15774.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 indexed asset_id, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetRemoved(opts *bind.WatchOpts, sink chan<- *BridgeAssetRemoved, asset_source []common.Address, asset_id []*big.Int) (event.Subscription, error) {

	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Removed", asset_sourceRule, asset_idRule)
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

// ParseAssetRemoved is a log parse operation binding the contract event 0x8b7f47a4e62d4fffe89da8d5381345e897518068150bd10f0903d0a722c15774.
//
// Solidity: event Asset_Removed(address indexed asset_source, uint256 indexed asset_id, uint256 nonce)
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
	AssetId     *big.Int
	Amount      *big.Int
	Nonce       *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterAssetWithdrawn is a free log retrieval operation binding the contract event 0x1d73f5f45122fcc00317f4af119d2a65f2dbaa07d7c023a0050b1fba5503365a.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 indexed asset_id, uint256 amount, uint256 nonce)
func (_Bridge *BridgeFilterer) FilterAssetWithdrawn(opts *bind.FilterOpts, user_address []common.Address, asset_source []common.Address, asset_id []*big.Int) (*BridgeAssetWithdrawnIterator, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Asset_Withdrawn", user_addressRule, asset_sourceRule, asset_idRule)
	if err != nil {
		return nil, err
	}
	return &BridgeAssetWithdrawnIterator{contract: _Bridge.contract, event: "Asset_Withdrawn", logs: logs, sub: sub}, nil
}

// WatchAssetWithdrawn is a free log subscription operation binding the contract event 0x1d73f5f45122fcc00317f4af119d2a65f2dbaa07d7c023a0050b1fba5503365a.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 indexed asset_id, uint256 amount, uint256 nonce)
func (_Bridge *BridgeFilterer) WatchAssetWithdrawn(opts *bind.WatchOpts, sink chan<- *BridgeAssetWithdrawn, user_address []common.Address, asset_source []common.Address, asset_id []*big.Int) (event.Subscription, error) {

	var user_addressRule []interface{}
	for _, user_addressItem := range user_address {
		user_addressRule = append(user_addressRule, user_addressItem)
	}
	var asset_sourceRule []interface{}
	for _, asset_sourceItem := range asset_source {
		asset_sourceRule = append(asset_sourceRule, asset_sourceItem)
	}
	var asset_idRule []interface{}
	for _, asset_idItem := range asset_id {
		asset_idRule = append(asset_idRule, asset_idItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Asset_Withdrawn", user_addressRule, asset_sourceRule, asset_idRule)
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

// ParseAssetWithdrawn is a log parse operation binding the contract event 0x1d73f5f45122fcc00317f4af119d2a65f2dbaa07d7c023a0050b1fba5503365a.
//
// Solidity: event Asset_Withdrawn(address indexed user_address, address indexed asset_source, uint256 indexed asset_id, uint256 amount, uint256 nonce)
func (_Bridge *BridgeFilterer) ParseAssetWithdrawn(log types.Log) (*BridgeAssetWithdrawn, error) {
	event := new(BridgeAssetWithdrawn)
	if err := _Bridge.contract.UnpackLog(event, "Asset_Withdrawn", log); err != nil {
		return nil, err
	}
	return event, nil
}

// BridgeMultisigControlSetIterator is returned from FilterMultisigControlSet and is used to iterate over the raw logs and unpacked data for MultisigControlSet events raised by the Bridge contract.
type BridgeMultisigControlSetIterator struct {
	Event *BridgeMultisigControlSet // Event containing the contract specifics and raw log

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
func (it *BridgeMultisigControlSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeMultisigControlSet)
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
		it.Event = new(BridgeMultisigControlSet)
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
func (it *BridgeMultisigControlSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeMultisigControlSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeMultisigControlSet represents a MultisigControlSet event raised by the Bridge contract.
type BridgeMultisigControlSet struct {
	MultisigControlSource common.Address
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterMultisigControlSet is a free log retrieval operation binding the contract event 0x1143e675ad794f5bd81a05b165b166be3a4e91f17d065f08809d88cefbd65406.
//
// Solidity: event Multisig_Control_Set(address indexed multisig_control_source)
func (_Bridge *BridgeFilterer) FilterMultisigControlSet(opts *bind.FilterOpts, multisig_control_source []common.Address) (*BridgeMultisigControlSetIterator, error) {

	var multisig_control_sourceRule []interface{}
	for _, multisig_control_sourceItem := range multisig_control_source {
		multisig_control_sourceRule = append(multisig_control_sourceRule, multisig_control_sourceItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "Multisig_Control_Set", multisig_control_sourceRule)
	if err != nil {
		return nil, err
	}
	return &BridgeMultisigControlSetIterator{contract: _Bridge.contract, event: "Multisig_Control_Set", logs: logs, sub: sub}, nil
}

// WatchMultisigControlSet is a free log subscription operation binding the contract event 0x1143e675ad794f5bd81a05b165b166be3a4e91f17d065f08809d88cefbd65406.
//
// Solidity: event Multisig_Control_Set(address indexed multisig_control_source)
func (_Bridge *BridgeFilterer) WatchMultisigControlSet(opts *bind.WatchOpts, sink chan<- *BridgeMultisigControlSet, multisig_control_source []common.Address) (event.Subscription, error) {

	var multisig_control_sourceRule []interface{}
	for _, multisig_control_sourceItem := range multisig_control_source {
		multisig_control_sourceRule = append(multisig_control_sourceRule, multisig_control_sourceItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "Multisig_Control_Set", multisig_control_sourceRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeMultisigControlSet)
				if err := _Bridge.contract.UnpackLog(event, "Multisig_Control_Set", log); err != nil {
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

// ParseMultisigControlSet is a log parse operation binding the contract event 0x1143e675ad794f5bd81a05b165b166be3a4e91f17d065f08809d88cefbd65406.
//
// Solidity: event Multisig_Control_Set(address indexed multisig_control_source)
func (_Bridge *BridgeFilterer) ParseMultisigControlSet(log types.Log) (*BridgeMultisigControlSet, error) {
	event := new(BridgeMultisigControlSet)
	if err := _Bridge.contract.UnpackLog(event, "Multisig_Control_Set", log); err != nil {
		return nil, err
	}
	return event, nil
}

// BridgeOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Bridge contract.
type BridgeOwnershipTransferredIterator struct {
	Event *BridgeOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *BridgeOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeOwnershipTransferred)
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
		it.Event = new(BridgeOwnershipTransferred)
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
func (it *BridgeOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeOwnershipTransferred represents a OwnershipTransferred event raised by the Bridge contract.
type BridgeOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Bridge *BridgeFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*BridgeOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Bridge.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BridgeOwnershipTransferredIterator{contract: _Bridge.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Bridge *BridgeFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BridgeOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Bridge.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeOwnershipTransferred)
				if err := _Bridge.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Bridge *BridgeFilterer) ParseOwnershipTransferred(log types.Log) (*BridgeOwnershipTransferred, error) {
	event := new(BridgeOwnershipTransferred)
	if err := _Bridge.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	return event, nil
}
