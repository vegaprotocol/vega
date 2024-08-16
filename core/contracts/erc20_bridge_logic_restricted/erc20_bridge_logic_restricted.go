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
	ABI: "[{\"type\":\"function\",\"name\":\"depositAsset\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"vegaPublicKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"exemptDepositor\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getAssetDepositLifetimeLimit\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAssetSource\",\"inputs\":[{\"name\":\"vegaAssetId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getMultisigControlAddress\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getVegaAssetId\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getWithdrawThreshold\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"globalResume\",\"inputs\":[{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"globalStop\",\"inputs\":[{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isAssetListed\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isExemptDepositor\",\"inputs\":[{\"name\":\"depositor\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"listAsset\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"vegaAssetId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"lifetimeLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"withdrawThreshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"removeAsset\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revokeExemptDepositor\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setAssetLimits\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"lifetimeLimit\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"threshold\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setWithdrawDelay\",\"inputs\":[{\"name\":\"delay\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawAsset\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"target\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"creation\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"AssetDeposited\",\"inputs\":[{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"assetSource\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"vegaPublicKey\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"AssetLimitsUpdated\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"lifetimeLimit\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"withdrawThreshold\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"AssetListed\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"vegaAssetId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"AssetRemoved\",\"inputs\":[{\"name\":\"assetSource\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"AssetWithdrawn\",\"inputs\":[{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"assetSource\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BridgeResumed\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BridgeStopped\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"BridgeWithdrawDelaySet\",\"inputs\":[{\"name\":\"withdraw_delay\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositorExempted\",\"inputs\":[{\"name\":\"depositor\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositorExemptionRevoked\",\"inputs\":[{\"name\":\"depositor\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false}]",
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

// GetAssetDepositLifetimeLimit is a free data retrieval call binding the contract method 0x5d1e1a73.
//
// Solidity: function getAssetDepositLifetimeLimit(address assetSource) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetAssetDepositLifetimeLimit(opts *bind.CallOpts, assetSource common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "getAssetDepositLifetimeLimit", assetSource)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAssetDepositLifetimeLimit is a free data retrieval call binding the contract method 0x5d1e1a73.
//
// Solidity: function getAssetDepositLifetimeLimit(address assetSource) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetAssetDepositLifetimeLimit(assetSource common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetDepositLifetimeLimit(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// GetAssetDepositLifetimeLimit is a free data retrieval call binding the contract method 0x5d1e1a73.
//
// Solidity: function getAssetDepositLifetimeLimit(address assetSource) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetAssetDepositLifetimeLimit(assetSource common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetDepositLifetimeLimit(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// GetAssetSource is a free data retrieval call binding the contract method 0xb653e56d.
//
// Solidity: function getAssetSource(bytes32 vegaAssetId) view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetAssetSource(opts *bind.CallOpts, vegaAssetId [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "getAssetSource", vegaAssetId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAssetSource is a free data retrieval call binding the contract method 0xb653e56d.
//
// Solidity: function getAssetSource(bytes32 vegaAssetId) view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetAssetSource(vegaAssetId [32]byte) (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetSource(&_Erc20BridgeLogicRestricted.CallOpts, vegaAssetId)
}

// GetAssetSource is a free data retrieval call binding the contract method 0xb653e56d.
//
// Solidity: function getAssetSource(bytes32 vegaAssetId) view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetAssetSource(vegaAssetId [32]byte) (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetAssetSource(&_Erc20BridgeLogicRestricted.CallOpts, vegaAssetId)
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0x81a8915e.
//
// Solidity: function getMultisigControlAddress() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetMultisigControlAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "getMultisigControlAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0x81a8915e.
//
// Solidity: function getMultisigControlAddress() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetMultisigControlAddress() (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetMultisigControlAddress(&_Erc20BridgeLogicRestricted.CallOpts)
}

// GetMultisigControlAddress is a free data retrieval call binding the contract method 0x81a8915e.
//
// Solidity: function getMultisigControlAddress() view returns(address)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetMultisigControlAddress() (common.Address, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetMultisigControlAddress(&_Erc20BridgeLogicRestricted.CallOpts)
}

// GetVegaAssetId is a free data retrieval call binding the contract method 0xf8c2dbe0.
//
// Solidity: function getVegaAssetId(address assetSource) view returns(bytes32)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetVegaAssetId(opts *bind.CallOpts, assetSource common.Address) ([32]byte, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "getVegaAssetId", assetSource)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetVegaAssetId is a free data retrieval call binding the contract method 0xf8c2dbe0.
//
// Solidity: function getVegaAssetId(address assetSource) view returns(bytes32)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetVegaAssetId(assetSource common.Address) ([32]byte, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetVegaAssetId(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// GetVegaAssetId is a free data retrieval call binding the contract method 0xf8c2dbe0.
//
// Solidity: function getVegaAssetId(address assetSource) view returns(bytes32)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetVegaAssetId(assetSource common.Address) ([32]byte, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetVegaAssetId(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// GetWithdrawThreshold is a free data retrieval call binding the contract method 0xf3dd4013.
//
// Solidity: function getWithdrawThreshold(address assetSource) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) GetWithdrawThreshold(opts *bind.CallOpts, assetSource common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "getWithdrawThreshold", assetSource)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWithdrawThreshold is a free data retrieval call binding the contract method 0xf3dd4013.
//
// Solidity: function getWithdrawThreshold(address assetSource) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GetWithdrawThreshold(assetSource common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetWithdrawThreshold(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// GetWithdrawThreshold is a free data retrieval call binding the contract method 0xf3dd4013.
//
// Solidity: function getWithdrawThreshold(address assetSource) view returns(uint256)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) GetWithdrawThreshold(assetSource common.Address) (*big.Int, error) {
	return _Erc20BridgeLogicRestricted.Contract.GetWithdrawThreshold(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// IsAssetListed is a free data retrieval call binding the contract method 0x47eaef01.
//
// Solidity: function isAssetListed(address assetSource) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) IsAssetListed(opts *bind.CallOpts, assetSource common.Address) (bool, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "isAssetListed", assetSource)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsAssetListed is a free data retrieval call binding the contract method 0x47eaef01.
//
// Solidity: function isAssetListed(address assetSource) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) IsAssetListed(assetSource common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsAssetListed(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// IsAssetListed is a free data retrieval call binding the contract method 0x47eaef01.
//
// Solidity: function isAssetListed(address assetSource) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) IsAssetListed(assetSource common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsAssetListed(&_Erc20BridgeLogicRestricted.CallOpts, assetSource)
}

// IsExemptDepositor is a free data retrieval call binding the contract method 0xe275317e.
//
// Solidity: function isExemptDepositor(address depositor) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCaller) IsExemptDepositor(opts *bind.CallOpts, depositor common.Address) (bool, error) {
	var out []interface{}
	err := _Erc20BridgeLogicRestricted.contract.Call(opts, &out, "isExemptDepositor", depositor)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsExemptDepositor is a free data retrieval call binding the contract method 0xe275317e.
//
// Solidity: function isExemptDepositor(address depositor) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) IsExemptDepositor(depositor common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsExemptDepositor(&_Erc20BridgeLogicRestricted.CallOpts, depositor)
}

// IsExemptDepositor is a free data retrieval call binding the contract method 0xe275317e.
//
// Solidity: function isExemptDepositor(address depositor) view returns(bool)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedCallerSession) IsExemptDepositor(depositor common.Address) (bool, error) {
	return _Erc20BridgeLogicRestricted.Contract.IsExemptDepositor(&_Erc20BridgeLogicRestricted.CallOpts, depositor)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xa372b1d2.
//
// Solidity: function depositAsset(address assetSource, uint256 amount, bytes32 vegaPublicKey) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) DepositAsset(opts *bind.TransactOpts, assetSource common.Address, amount *big.Int, vegaPublicKey [32]byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "depositAsset", assetSource, amount, vegaPublicKey)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xa372b1d2.
//
// Solidity: function depositAsset(address assetSource, uint256 amount, bytes32 vegaPublicKey) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) DepositAsset(assetSource common.Address, amount *big.Int, vegaPublicKey [32]byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.DepositAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, amount, vegaPublicKey)
}

// DepositAsset is a paid mutator transaction binding the contract method 0xa372b1d2.
//
// Solidity: function depositAsset(address assetSource, uint256 amount, bytes32 vegaPublicKey) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) DepositAsset(assetSource common.Address, amount *big.Int, vegaPublicKey [32]byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.DepositAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, amount, vegaPublicKey)
}

// ExemptDepositor is a paid mutator transaction binding the contract method 0xfc4e5482.
//
// Solidity: function exemptDepositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) ExemptDepositor(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "exemptDepositor")
}

// ExemptDepositor is a paid mutator transaction binding the contract method 0xfc4e5482.
//
// Solidity: function exemptDepositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) ExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// ExemptDepositor is a paid mutator transaction binding the contract method 0xfc4e5482.
//
// Solidity: function exemptDepositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) ExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// GlobalResume is a paid mutator transaction binding the contract method 0x3c00cb9c.
//
// Solidity: function globalResume(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) GlobalResume(opts *bind.TransactOpts, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "globalResume", nonce, signatures)
}

// GlobalResume is a paid mutator transaction binding the contract method 0x3c00cb9c.
//
// Solidity: function globalResume(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GlobalResume(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalResume(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// GlobalResume is a paid mutator transaction binding the contract method 0x3c00cb9c.
//
// Solidity: function globalResume(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) GlobalResume(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalResume(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// GlobalStop is a paid mutator transaction binding the contract method 0xdaff4e2f.
//
// Solidity: function globalStop(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) GlobalStop(opts *bind.TransactOpts, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "globalStop", nonce, signatures)
}

// GlobalStop is a paid mutator transaction binding the contract method 0xdaff4e2f.
//
// Solidity: function globalStop(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) GlobalStop(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalStop(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// GlobalStop is a paid mutator transaction binding the contract method 0xdaff4e2f.
//
// Solidity: function globalStop(uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) GlobalStop(nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.GlobalStop(&_Erc20BridgeLogicRestricted.TransactOpts, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x8ed131e0.
//
// Solidity: function listAsset(address assetSource, bytes32 vegaAssetId, uint256 lifetimeLimit, uint256 withdrawThreshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) ListAsset(opts *bind.TransactOpts, assetSource common.Address, vegaAssetId [32]byte, lifetimeLimit *big.Int, withdrawThreshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "listAsset", assetSource, vegaAssetId, lifetimeLimit, withdrawThreshold, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x8ed131e0.
//
// Solidity: function listAsset(address assetSource, bytes32 vegaAssetId, uint256 lifetimeLimit, uint256 withdrawThreshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) ListAsset(assetSource common.Address, vegaAssetId [32]byte, lifetimeLimit *big.Int, withdrawThreshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ListAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, vegaAssetId, lifetimeLimit, withdrawThreshold, nonce, signatures)
}

// ListAsset is a paid mutator transaction binding the contract method 0x8ed131e0.
//
// Solidity: function listAsset(address assetSource, bytes32 vegaAssetId, uint256 lifetimeLimit, uint256 withdrawThreshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) ListAsset(assetSource common.Address, vegaAssetId [32]byte, lifetimeLimit *big.Int, withdrawThreshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.ListAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, vegaAssetId, lifetimeLimit, withdrawThreshold, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0x8ef973ea.
//
// Solidity: function removeAsset(address assetSource, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) RemoveAsset(opts *bind.TransactOpts, assetSource common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "removeAsset", assetSource, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0x8ef973ea.
//
// Solidity: function removeAsset(address assetSource, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) RemoveAsset(assetSource common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RemoveAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, nonce, signatures)
}

// RemoveAsset is a paid mutator transaction binding the contract method 0x8ef973ea.
//
// Solidity: function removeAsset(address assetSource, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) RemoveAsset(assetSource common.Address, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RemoveAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, nonce, signatures)
}

// RevokeExemptDepositor is a paid mutator transaction binding the contract method 0x66a3edc4.
//
// Solidity: function revokeExemptDepositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) RevokeExemptDepositor(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "revokeExemptDepositor")
}

// RevokeExemptDepositor is a paid mutator transaction binding the contract method 0x66a3edc4.
//
// Solidity: function revokeExemptDepositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) RevokeExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RevokeExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// RevokeExemptDepositor is a paid mutator transaction binding the contract method 0x66a3edc4.
//
// Solidity: function revokeExemptDepositor() returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) RevokeExemptDepositor() (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.RevokeExemptDepositor(&_Erc20BridgeLogicRestricted.TransactOpts)
}

// SetAssetLimits is a paid mutator transaction binding the contract method 0x10725d39.
//
// Solidity: function setAssetLimits(address assetSource, uint256 lifetimeLimit, uint256 threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) SetAssetLimits(opts *bind.TransactOpts, assetSource common.Address, lifetimeLimit *big.Int, threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "setAssetLimits", assetSource, lifetimeLimit, threshold, nonce, signatures)
}

// SetAssetLimits is a paid mutator transaction binding the contract method 0x10725d39.
//
// Solidity: function setAssetLimits(address assetSource, uint256 lifetimeLimit, uint256 threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) SetAssetLimits(assetSource common.Address, lifetimeLimit *big.Int, threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetAssetLimits(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, lifetimeLimit, threshold, nonce, signatures)
}

// SetAssetLimits is a paid mutator transaction binding the contract method 0x10725d39.
//
// Solidity: function setAssetLimits(address assetSource, uint256 lifetimeLimit, uint256 threshold, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) SetAssetLimits(assetSource common.Address, lifetimeLimit *big.Int, threshold *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetAssetLimits(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, lifetimeLimit, threshold, nonce, signatures)
}

// SetWithdrawDelay is a paid mutator transaction binding the contract method 0x9fde4ad0.
//
// Solidity: function setWithdrawDelay(uint256 delay, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) SetWithdrawDelay(opts *bind.TransactOpts, delay *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "setWithdrawDelay", delay, nonce, signatures)
}

// SetWithdrawDelay is a paid mutator transaction binding the contract method 0x9fde4ad0.
//
// Solidity: function setWithdrawDelay(uint256 delay, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) SetWithdrawDelay(delay *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetWithdrawDelay(&_Erc20BridgeLogicRestricted.TransactOpts, delay, nonce, signatures)
}

// SetWithdrawDelay is a paid mutator transaction binding the contract method 0x9fde4ad0.
//
// Solidity: function setWithdrawDelay(uint256 delay, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) SetWithdrawDelay(delay *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.SetWithdrawDelay(&_Erc20BridgeLogicRestricted.TransactOpts, delay, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x13b99c74.
//
// Solidity: function withdrawAsset(address assetSource, uint256 amount, address target, uint256 creation, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactor) WithdrawAsset(opts *bind.TransactOpts, assetSource common.Address, amount *big.Int, target common.Address, creation *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.contract.Transact(opts, "withdrawAsset", assetSource, amount, target, creation, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x13b99c74.
//
// Solidity: function withdrawAsset(address assetSource, uint256 amount, address target, uint256 creation, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedSession) WithdrawAsset(assetSource common.Address, amount *big.Int, target common.Address, creation *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.WithdrawAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, amount, target, creation, nonce, signatures)
}

// WithdrawAsset is a paid mutator transaction binding the contract method 0x13b99c74.
//
// Solidity: function withdrawAsset(address assetSource, uint256 amount, address target, uint256 creation, uint256 nonce, bytes signatures) returns()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedTransactorSession) WithdrawAsset(assetSource common.Address, amount *big.Int, target common.Address, creation *big.Int, nonce *big.Int, signatures []byte) (*types.Transaction, error) {
	return _Erc20BridgeLogicRestricted.Contract.WithdrawAsset(&_Erc20BridgeLogicRestricted.TransactOpts, assetSource, amount, target, creation, nonce, signatures)
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

// FilterAssetDeposited is a free log retrieval operation binding the contract event 0x682eab6abd797cfe9f53779827c6526e4e1e5c2c0ad2c2a30d8af5a269f5608e.
//
// Solidity: event AssetDeposited(address indexed userAddress, address indexed assetSource, uint256 amount, bytes32 vegaPublicKey)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetDeposited(opts *bind.FilterOpts, userAddress []common.Address, assetSource []common.Address) (*Erc20BridgeLogicRestrictedAssetDepositedIterator, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "AssetDeposited", userAddressRule, assetSourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetDepositedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "AssetDeposited", logs: logs, sub: sub}, nil
}

// WatchAssetDeposited is a free log subscription operation binding the contract event 0x682eab6abd797cfe9f53779827c6526e4e1e5c2c0ad2c2a30d8af5a269f5608e.
//
// Solidity: event AssetDeposited(address indexed userAddress, address indexed assetSource, uint256 amount, bytes32 vegaPublicKey)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetDeposited(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetDeposited, userAddress []common.Address, assetSource []common.Address) (event.Subscription, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "AssetDeposited", userAddressRule, assetSourceRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetDeposited", log); err != nil {
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

// ParseAssetDeposited is a log parse operation binding the contract event 0x682eab6abd797cfe9f53779827c6526e4e1e5c2c0ad2c2a30d8af5a269f5608e.
//
// Solidity: event AssetDeposited(address indexed userAddress, address indexed assetSource, uint256 amount, bytes32 vegaPublicKey)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetDeposited(log types.Log) (*Erc20BridgeLogicRestrictedAssetDeposited, error) {
	event := new(Erc20BridgeLogicRestrictedAssetDeposited)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetDeposited", log); err != nil {
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

// FilterAssetLimitsUpdated is a free log retrieval operation binding the contract event 0x2b9be4c285a9f5ff31bc13fabbab637c16e5f945e41a8a0e00e699264e9a17b6.
//
// Solidity: event AssetLimitsUpdated(address indexed assetSource, uint256 lifetimeLimit, uint256 withdrawThreshold)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetLimitsUpdated(opts *bind.FilterOpts, assetSource []common.Address) (*Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator, error) {

	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "AssetLimitsUpdated", assetSourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetLimitsUpdatedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "AssetLimitsUpdated", logs: logs, sub: sub}, nil
}

// WatchAssetLimitsUpdated is a free log subscription operation binding the contract event 0x2b9be4c285a9f5ff31bc13fabbab637c16e5f945e41a8a0e00e699264e9a17b6.
//
// Solidity: event AssetLimitsUpdated(address indexed assetSource, uint256 lifetimeLimit, uint256 withdrawThreshold)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetLimitsUpdated(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetLimitsUpdated, assetSource []common.Address) (event.Subscription, error) {

	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "AssetLimitsUpdated", assetSourceRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetLimitsUpdated", log); err != nil {
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

// ParseAssetLimitsUpdated is a log parse operation binding the contract event 0x2b9be4c285a9f5ff31bc13fabbab637c16e5f945e41a8a0e00e699264e9a17b6.
//
// Solidity: event AssetLimitsUpdated(address indexed assetSource, uint256 lifetimeLimit, uint256 withdrawThreshold)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetLimitsUpdated(log types.Log) (*Erc20BridgeLogicRestrictedAssetLimitsUpdated, error) {
	event := new(Erc20BridgeLogicRestrictedAssetLimitsUpdated)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetLimitsUpdated", log); err != nil {
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

// FilterAssetListed is a free log retrieval operation binding the contract event 0xec587e0b31172eee3a953d82b0c7f78c96be78084a92eb0463144c0446250d49.
//
// Solidity: event AssetListed(address indexed assetSource, bytes32 indexed vegaAssetId, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetListed(opts *bind.FilterOpts, assetSource []common.Address, vegaAssetId [][32]byte) (*Erc20BridgeLogicRestrictedAssetListedIterator, error) {

	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}
	var vegaAssetIdRule []interface{}
	for _, vegaAssetIdItem := range vegaAssetId {
		vegaAssetIdRule = append(vegaAssetIdRule, vegaAssetIdItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "AssetListed", assetSourceRule, vegaAssetIdRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetListedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "AssetListed", logs: logs, sub: sub}, nil
}

// WatchAssetListed is a free log subscription operation binding the contract event 0xec587e0b31172eee3a953d82b0c7f78c96be78084a92eb0463144c0446250d49.
//
// Solidity: event AssetListed(address indexed assetSource, bytes32 indexed vegaAssetId, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetListed(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetListed, assetSource []common.Address, vegaAssetId [][32]byte) (event.Subscription, error) {

	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}
	var vegaAssetIdRule []interface{}
	for _, vegaAssetIdItem := range vegaAssetId {
		vegaAssetIdRule = append(vegaAssetIdRule, vegaAssetIdItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "AssetListed", assetSourceRule, vegaAssetIdRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetListed", log); err != nil {
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

// ParseAssetListed is a log parse operation binding the contract event 0xec587e0b31172eee3a953d82b0c7f78c96be78084a92eb0463144c0446250d49.
//
// Solidity: event AssetListed(address indexed assetSource, bytes32 indexed vegaAssetId, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetListed(log types.Log) (*Erc20BridgeLogicRestrictedAssetListed, error) {
	event := new(Erc20BridgeLogicRestrictedAssetListed)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetListed", log); err != nil {
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

// FilterAssetRemoved is a free log retrieval operation binding the contract event 0x3406221f53114f44c9a1bb93d08ee55735f39bf235a54741684a52501207bb54.
//
// Solidity: event AssetRemoved(address indexed assetSource, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetRemoved(opts *bind.FilterOpts, assetSource []common.Address) (*Erc20BridgeLogicRestrictedAssetRemovedIterator, error) {

	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "AssetRemoved", assetSourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetRemovedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "AssetRemoved", logs: logs, sub: sub}, nil
}

// WatchAssetRemoved is a free log subscription operation binding the contract event 0x3406221f53114f44c9a1bb93d08ee55735f39bf235a54741684a52501207bb54.
//
// Solidity: event AssetRemoved(address indexed assetSource, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetRemoved(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetRemoved, assetSource []common.Address) (event.Subscription, error) {

	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "AssetRemoved", assetSourceRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetRemoved", log); err != nil {
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

// ParseAssetRemoved is a log parse operation binding the contract event 0x3406221f53114f44c9a1bb93d08ee55735f39bf235a54741684a52501207bb54.
//
// Solidity: event AssetRemoved(address indexed assetSource, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetRemoved(log types.Log) (*Erc20BridgeLogicRestrictedAssetRemoved, error) {
	event := new(Erc20BridgeLogicRestrictedAssetRemoved)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetRemoved", log); err != nil {
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

// FilterAssetWithdrawn is a free log retrieval operation binding the contract event 0xdd3541b6a4a74daf4ad7d33c0d4b441e1bad2e93e12da692aa37c5febb19f7b8.
//
// Solidity: event AssetWithdrawn(address indexed userAddress, address indexed assetSource, uint256 amount, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterAssetWithdrawn(opts *bind.FilterOpts, userAddress []common.Address, assetSource []common.Address) (*Erc20BridgeLogicRestrictedAssetWithdrawnIterator, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "AssetWithdrawn", userAddressRule, assetSourceRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedAssetWithdrawnIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "AssetWithdrawn", logs: logs, sub: sub}, nil
}

// WatchAssetWithdrawn is a free log subscription operation binding the contract event 0xdd3541b6a4a74daf4ad7d33c0d4b441e1bad2e93e12da692aa37c5febb19f7b8.
//
// Solidity: event AssetWithdrawn(address indexed userAddress, address indexed assetSource, uint256 amount, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchAssetWithdrawn(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedAssetWithdrawn, userAddress []common.Address, assetSource []common.Address) (event.Subscription, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var assetSourceRule []interface{}
	for _, assetSourceItem := range assetSource {
		assetSourceRule = append(assetSourceRule, assetSourceItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "AssetWithdrawn", userAddressRule, assetSourceRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetWithdrawn", log); err != nil {
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

// ParseAssetWithdrawn is a log parse operation binding the contract event 0xdd3541b6a4a74daf4ad7d33c0d4b441e1bad2e93e12da692aa37c5febb19f7b8.
//
// Solidity: event AssetWithdrawn(address indexed userAddress, address indexed assetSource, uint256 amount, uint256 nonce)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseAssetWithdrawn(log types.Log) (*Erc20BridgeLogicRestrictedAssetWithdrawn, error) {
	event := new(Erc20BridgeLogicRestrictedAssetWithdrawn)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "AssetWithdrawn", log); err != nil {
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

// FilterBridgeResumed is a free log retrieval operation binding the contract event 0x287ac30ff6e68ce13e26be0638f2f8fb754a569fd9c2dc77bf8241411a647876.
//
// Solidity: event BridgeResumed()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterBridgeResumed(opts *bind.FilterOpts) (*Erc20BridgeLogicRestrictedBridgeResumedIterator, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "BridgeResumed")
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedBridgeResumedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "BridgeResumed", logs: logs, sub: sub}, nil
}

// WatchBridgeResumed is a free log subscription operation binding the contract event 0x287ac30ff6e68ce13e26be0638f2f8fb754a569fd9c2dc77bf8241411a647876.
//
// Solidity: event BridgeResumed()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchBridgeResumed(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedBridgeResumed) (event.Subscription, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "BridgeResumed")
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "BridgeResumed", log); err != nil {
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

// ParseBridgeResumed is a log parse operation binding the contract event 0x287ac30ff6e68ce13e26be0638f2f8fb754a569fd9c2dc77bf8241411a647876.
//
// Solidity: event BridgeResumed()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseBridgeResumed(log types.Log) (*Erc20BridgeLogicRestrictedBridgeResumed, error) {
	event := new(Erc20BridgeLogicRestrictedBridgeResumed)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "BridgeResumed", log); err != nil {
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

// FilterBridgeStopped is a free log retrieval operation binding the contract event 0xc491bbb4be1472096d01e3f0b6f3c2ba9720a559c3422c36408dff58e42d3873.
//
// Solidity: event BridgeStopped()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterBridgeStopped(opts *bind.FilterOpts) (*Erc20BridgeLogicRestrictedBridgeStoppedIterator, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "BridgeStopped")
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedBridgeStoppedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "BridgeStopped", logs: logs, sub: sub}, nil
}

// WatchBridgeStopped is a free log subscription operation binding the contract event 0xc491bbb4be1472096d01e3f0b6f3c2ba9720a559c3422c36408dff58e42d3873.
//
// Solidity: event BridgeStopped()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchBridgeStopped(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedBridgeStopped) (event.Subscription, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "BridgeStopped")
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "BridgeStopped", log); err != nil {
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

// ParseBridgeStopped is a log parse operation binding the contract event 0xc491bbb4be1472096d01e3f0b6f3c2ba9720a559c3422c36408dff58e42d3873.
//
// Solidity: event BridgeStopped()
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseBridgeStopped(log types.Log) (*Erc20BridgeLogicRestrictedBridgeStopped, error) {
	event := new(Erc20BridgeLogicRestrictedBridgeStopped)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "BridgeStopped", log); err != nil {
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

// FilterBridgeWithdrawDelaySet is a free log retrieval operation binding the contract event 0xd7ce7fbb38daf3a282293f127641ae35e6f17cee1b51904e7b9c633f97df3b85.
//
// Solidity: event BridgeWithdrawDelaySet(uint256 withdraw_delay)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterBridgeWithdrawDelaySet(opts *bind.FilterOpts) (*Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "BridgeWithdrawDelaySet")
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedBridgeWithdrawDelaySetIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "BridgeWithdrawDelaySet", logs: logs, sub: sub}, nil
}

// WatchBridgeWithdrawDelaySet is a free log subscription operation binding the contract event 0xd7ce7fbb38daf3a282293f127641ae35e6f17cee1b51904e7b9c633f97df3b85.
//
// Solidity: event BridgeWithdrawDelaySet(uint256 withdraw_delay)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchBridgeWithdrawDelaySet(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet) (event.Subscription, error) {

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "BridgeWithdrawDelaySet")
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "BridgeWithdrawDelaySet", log); err != nil {
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

// ParseBridgeWithdrawDelaySet is a log parse operation binding the contract event 0xd7ce7fbb38daf3a282293f127641ae35e6f17cee1b51904e7b9c633f97df3b85.
//
// Solidity: event BridgeWithdrawDelaySet(uint256 withdraw_delay)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseBridgeWithdrawDelaySet(log types.Log) (*Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet, error) {
	event := new(Erc20BridgeLogicRestrictedBridgeWithdrawDelaySet)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "BridgeWithdrawDelaySet", log); err != nil {
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

// FilterDepositorExempted is a free log retrieval operation binding the contract event 0x9864ffa3d25fb42ecb8e42e8c6655954e8c7427d44a7a9f8edfe0cea4a0108f8.
//
// Solidity: event DepositorExempted(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterDepositorExempted(opts *bind.FilterOpts, depositor []common.Address) (*Erc20BridgeLogicRestrictedDepositorExemptedIterator, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "DepositorExempted", depositorRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedDepositorExemptedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "DepositorExempted", logs: logs, sub: sub}, nil
}

// WatchDepositorExempted is a free log subscription operation binding the contract event 0x9864ffa3d25fb42ecb8e42e8c6655954e8c7427d44a7a9f8edfe0cea4a0108f8.
//
// Solidity: event DepositorExempted(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchDepositorExempted(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedDepositorExempted, depositor []common.Address) (event.Subscription, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "DepositorExempted", depositorRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "DepositorExempted", log); err != nil {
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

// ParseDepositorExempted is a log parse operation binding the contract event 0x9864ffa3d25fb42ecb8e42e8c6655954e8c7427d44a7a9f8edfe0cea4a0108f8.
//
// Solidity: event DepositorExempted(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseDepositorExempted(log types.Log) (*Erc20BridgeLogicRestrictedDepositorExempted, error) {
	event := new(Erc20BridgeLogicRestrictedDepositorExempted)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "DepositorExempted", log); err != nil {
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

// FilterDepositorExemptionRevoked is a free log retrieval operation binding the contract event 0xd661c54cb18e99cf2cc5b8c6b57f7a094e8a1b967b5de45a609f6d0220e998d4.
//
// Solidity: event DepositorExemptionRevoked(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) FilterDepositorExemptionRevoked(opts *bind.FilterOpts, depositor []common.Address) (*Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.FilterLogs(opts, "DepositorExemptionRevoked", depositorRule)
	if err != nil {
		return nil, err
	}
	return &Erc20BridgeLogicRestrictedDepositorExemptionRevokedIterator{contract: _Erc20BridgeLogicRestricted.contract, event: "DepositorExemptionRevoked", logs: logs, sub: sub}, nil
}

// WatchDepositorExemptionRevoked is a free log subscription operation binding the contract event 0xd661c54cb18e99cf2cc5b8c6b57f7a094e8a1b967b5de45a609f6d0220e998d4.
//
// Solidity: event DepositorExemptionRevoked(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) WatchDepositorExemptionRevoked(opts *bind.WatchOpts, sink chan<- *Erc20BridgeLogicRestrictedDepositorExemptionRevoked, depositor []common.Address) (event.Subscription, error) {

	var depositorRule []interface{}
	for _, depositorItem := range depositor {
		depositorRule = append(depositorRule, depositorItem)
	}

	logs, sub, err := _Erc20BridgeLogicRestricted.contract.WatchLogs(opts, "DepositorExemptionRevoked", depositorRule)
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
				if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "DepositorExemptionRevoked", log); err != nil {
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

// ParseDepositorExemptionRevoked is a log parse operation binding the contract event 0xd661c54cb18e99cf2cc5b8c6b57f7a094e8a1b967b5de45a609f6d0220e998d4.
//
// Solidity: event DepositorExemptionRevoked(address indexed depositor)
func (_Erc20BridgeLogicRestricted *Erc20BridgeLogicRestrictedFilterer) ParseDepositorExemptionRevoked(log types.Log) (*Erc20BridgeLogicRestrictedDepositorExemptionRevoked, error) {
	event := new(Erc20BridgeLogicRestrictedDepositorExemptionRevoked)
	if err := _Erc20BridgeLogicRestricted.contract.UnpackLog(event, "DepositorExemptionRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
