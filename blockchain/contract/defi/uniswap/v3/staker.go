// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package uniswapv3

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

// IUniswapV3StakerIncentiveKey is an auto generated low-level Go binding around an user-defined struct.
type IUniswapV3StakerIncentiveKey struct {
	RewardToken common.Address
	Pool        common.Address
	StartTime   *big.Int
	EndTime     *big.Int
	Refundee    common.Address
}

// StakerMetaData contains all meta data concerning the Staker contract.
var StakerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIUniswapV3Factory\",\"name\":\"_factory\",\"type\":\"address\"},{\"internalType\":\"contractINonfungiblePositionManager\",\"name\":\"_nonfungiblePositionManager\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_maxIncentiveStartLeadTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_maxIncentiveDuration\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"oldOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"DepositTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"contractIUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"startTime\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"refundee\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"reward\",\"type\":\"uint256\"}],\"name\":\"IncentiveCreated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"incentiveId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"refund\",\"type\":\"uint256\"}],\"name\":\"IncentiveEnded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"reward\",\"type\":\"uint256\"}],\"name\":\"RewardClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"incentiveId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"}],\"name\":\"TokenStaked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"incentiveId\",\"type\":\"bytes32\"}],\"name\":\"TokenUnstaked\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amountRequested\",\"type\":\"uint256\"}],\"name\":\"claimReward\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"reward\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"internalType\":\"contractIUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"startTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"refundee\",\"type\":\"address\"}],\"internalType\":\"structIUniswapV3Staker.IncentiveKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"reward\",\"type\":\"uint256\"}],\"name\":\"createIncentive\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint48\",\"name\":\"numberOfStakes\",\"type\":\"uint48\"},{\"internalType\":\"int24\",\"name\":\"tickLower\",\"type\":\"int24\"},{\"internalType\":\"int24\",\"name\":\"tickUpper\",\"type\":\"int24\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"internalType\":\"contractIUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"startTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"refundee\",\"type\":\"address\"}],\"internalType\":\"structIUniswapV3Staker.IncentiveKey\",\"name\":\"key\",\"type\":\"tuple\"}],\"name\":\"endIncentive\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"refund\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"contractIUniswapV3Factory\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"internalType\":\"contractIUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"startTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"refundee\",\"type\":\"address\"}],\"internalType\":\"structIUniswapV3Staker.IncentiveKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"getRewardInfo\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"reward\",\"type\":\"uint256\"},{\"internalType\":\"uint160\",\"name\":\"secondsInsideX128\",\"type\":\"uint160\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"incentives\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"totalRewardUnclaimed\",\"type\":\"uint256\"},{\"internalType\":\"uint160\",\"name\":\"totalSecondsClaimedX128\",\"type\":\"uint160\"},{\"internalType\":\"uint96\",\"name\":\"numberOfStakes\",\"type\":\"uint96\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxIncentiveDuration\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxIncentiveStartLeadTime\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes[]\",\"name\":\"data\",\"type\":\"bytes[]\"}],\"name\":\"multicall\",\"outputs\":[{\"internalType\":\"bytes[]\",\"name\":\"results\",\"type\":\"bytes[]\"}],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nonfungiblePositionManager\",\"outputs\":[{\"internalType\":\"contractINonfungiblePositionManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"onERC721Received\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"rewards\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"internalType\":\"contractIUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"startTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"refundee\",\"type\":\"address\"}],\"internalType\":\"structIUniswapV3Staker.IncentiveKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"stakeToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"incentiveId\",\"type\":\"bytes32\"}],\"name\":\"stakes\",\"outputs\":[{\"internalType\":\"uint160\",\"name\":\"secondsPerLiquidityInsideInitialX128\",\"type\":\"uint160\"},{\"internalType\":\"uint128\",\"name\":\"liquidity\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"transferDeposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"contractIERC20Minimal\",\"name\":\"rewardToken\",\"type\":\"address\"},{\"internalType\":\"contractIUniswapV3Pool\",\"name\":\"pool\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"startTime\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"endTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"refundee\",\"type\":\"address\"}],\"internalType\":\"structIUniswapV3Staker.IncentiveKey\",\"name\":\"key\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"}],\"name\":\"unstakeToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"tokenId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"withdrawToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// StakerABI is the input ABI used to generate the binding from.
// Deprecated: Use StakerMetaData.ABI instead.
var StakerABI = StakerMetaData.ABI

// Staker is an auto generated Go binding around an Ethereum contract.
type Staker struct {
	StakerCaller     // Read-only binding to the contract
	StakerTransactor // Write-only binding to the contract
	StakerFilterer   // Log filterer for contract events
}

// StakerCaller is an auto generated read-only Go binding around an Ethereum contract.
type StakerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StakerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StakerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StakerSession struct {
	Contract     *Staker           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StakerCallerSession struct {
	Contract *StakerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// StakerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StakerTransactorSession struct {
	Contract     *StakerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakerRaw is an auto generated low-level Go binding around an Ethereum contract.
type StakerRaw struct {
	Contract *Staker // Generic contract binding to access the raw methods on
}

// StakerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StakerCallerRaw struct {
	Contract *StakerCaller // Generic read-only contract binding to access the raw methods on
}

// StakerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StakerTransactorRaw struct {
	Contract *StakerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStaker creates a new instance of Staker, bound to a specific deployed contract.
func NewStaker(address common.Address, backend bind.ContractBackend) (*Staker, error) {
	contract, err := bindStaker(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Staker{StakerCaller: StakerCaller{contract: contract}, StakerTransactor: StakerTransactor{contract: contract}, StakerFilterer: StakerFilterer{contract: contract}}, nil
}

// NewStakerCaller creates a new read-only instance of Staker, bound to a specific deployed contract.
func NewStakerCaller(address common.Address, caller bind.ContractCaller) (*StakerCaller, error) {
	contract, err := bindStaker(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StakerCaller{contract: contract}, nil
}

// NewStakerTransactor creates a new write-only instance of Staker, bound to a specific deployed contract.
func NewStakerTransactor(address common.Address, transactor bind.ContractTransactor) (*StakerTransactor, error) {
	contract, err := bindStaker(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StakerTransactor{contract: contract}, nil
}

// NewStakerFilterer creates a new log filterer instance of Staker, bound to a specific deployed contract.
func NewStakerFilterer(address common.Address, filterer bind.ContractFilterer) (*StakerFilterer, error) {
	contract, err := bindStaker(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StakerFilterer{contract: contract}, nil
}

// bindStaker binds a generic wrapper to an already deployed contract.
func bindStaker(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StakerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Staker *StakerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Staker.Contract.StakerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Staker *StakerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staker.Contract.StakerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Staker *StakerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Staker.Contract.StakerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Staker *StakerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Staker.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Staker *StakerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staker.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Staker *StakerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Staker.Contract.contract.Transact(opts, method, params...)
}

// Deposits is a free data retrieval call binding the contract method 0xb02c43d0.
//
// Solidity: function deposits(uint256 ) view returns(address owner, uint48 numberOfStakes, int24 tickLower, int24 tickUpper)
func (_Staker *StakerCaller) Deposits(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Owner          common.Address
	NumberOfStakes *big.Int
	TickLower      *big.Int
	TickUpper      *big.Int
}, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "deposits", arg0)

	outstruct := new(struct {
		Owner          common.Address
		NumberOfStakes *big.Int
		TickLower      *big.Int
		TickUpper      *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Owner = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.NumberOfStakes = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TickLower = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.TickUpper = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Deposits is a free data retrieval call binding the contract method 0xb02c43d0.
//
// Solidity: function deposits(uint256 ) view returns(address owner, uint48 numberOfStakes, int24 tickLower, int24 tickUpper)
func (_Staker *StakerSession) Deposits(arg0 *big.Int) (struct {
	Owner          common.Address
	NumberOfStakes *big.Int
	TickLower      *big.Int
	TickUpper      *big.Int
}, error) {
	return _Staker.Contract.Deposits(&_Staker.CallOpts, arg0)
}

// Deposits is a free data retrieval call binding the contract method 0xb02c43d0.
//
// Solidity: function deposits(uint256 ) view returns(address owner, uint48 numberOfStakes, int24 tickLower, int24 tickUpper)
func (_Staker *StakerCallerSession) Deposits(arg0 *big.Int) (struct {
	Owner          common.Address
	NumberOfStakes *big.Int
	TickLower      *big.Int
	TickUpper      *big.Int
}, error) {
	return _Staker.Contract.Deposits(&_Staker.CallOpts, arg0)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Staker *StakerCaller) Factory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "factory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Staker *StakerSession) Factory() (common.Address, error) {
	return _Staker.Contract.Factory(&_Staker.CallOpts)
}

// Factory is a free data retrieval call binding the contract method 0xc45a0155.
//
// Solidity: function factory() view returns(address)
func (_Staker *StakerCallerSession) Factory() (common.Address, error) {
	return _Staker.Contract.Factory(&_Staker.CallOpts)
}

// GetRewardInfo is a free data retrieval call binding the contract method 0xd953186e.
//
// Solidity: function getRewardInfo((address,address,uint256,uint256,address) key, uint256 tokenId) view returns(uint256 reward, uint160 secondsInsideX128)
func (_Staker *StakerCaller) GetRewardInfo(opts *bind.CallOpts, key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (struct {
	Reward            *big.Int
	SecondsInsideX128 *big.Int
}, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "getRewardInfo", key, tokenId)

	outstruct := new(struct {
		Reward            *big.Int
		SecondsInsideX128 *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Reward = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.SecondsInsideX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetRewardInfo is a free data retrieval call binding the contract method 0xd953186e.
//
// Solidity: function getRewardInfo((address,address,uint256,uint256,address) key, uint256 tokenId) view returns(uint256 reward, uint160 secondsInsideX128)
func (_Staker *StakerSession) GetRewardInfo(key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (struct {
	Reward            *big.Int
	SecondsInsideX128 *big.Int
}, error) {
	return _Staker.Contract.GetRewardInfo(&_Staker.CallOpts, key, tokenId)
}

// GetRewardInfo is a free data retrieval call binding the contract method 0xd953186e.
//
// Solidity: function getRewardInfo((address,address,uint256,uint256,address) key, uint256 tokenId) view returns(uint256 reward, uint160 secondsInsideX128)
func (_Staker *StakerCallerSession) GetRewardInfo(key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (struct {
	Reward            *big.Int
	SecondsInsideX128 *big.Int
}, error) {
	return _Staker.Contract.GetRewardInfo(&_Staker.CallOpts, key, tokenId)
}

// Incentives is a free data retrieval call binding the contract method 0x60777795.
//
// Solidity: function incentives(bytes32 ) view returns(uint256 totalRewardUnclaimed, uint160 totalSecondsClaimedX128, uint96 numberOfStakes)
func (_Staker *StakerCaller) Incentives(opts *bind.CallOpts, arg0 [32]byte) (struct {
	TotalRewardUnclaimed    *big.Int
	TotalSecondsClaimedX128 *big.Int
	NumberOfStakes          *big.Int
}, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "incentives", arg0)

	outstruct := new(struct {
		TotalRewardUnclaimed    *big.Int
		TotalSecondsClaimedX128 *big.Int
		NumberOfStakes          *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.TotalRewardUnclaimed = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.TotalSecondsClaimedX128 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.NumberOfStakes = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Incentives is a free data retrieval call binding the contract method 0x60777795.
//
// Solidity: function incentives(bytes32 ) view returns(uint256 totalRewardUnclaimed, uint160 totalSecondsClaimedX128, uint96 numberOfStakes)
func (_Staker *StakerSession) Incentives(arg0 [32]byte) (struct {
	TotalRewardUnclaimed    *big.Int
	TotalSecondsClaimedX128 *big.Int
	NumberOfStakes          *big.Int
}, error) {
	return _Staker.Contract.Incentives(&_Staker.CallOpts, arg0)
}

// Incentives is a free data retrieval call binding the contract method 0x60777795.
//
// Solidity: function incentives(bytes32 ) view returns(uint256 totalRewardUnclaimed, uint160 totalSecondsClaimedX128, uint96 numberOfStakes)
func (_Staker *StakerCallerSession) Incentives(arg0 [32]byte) (struct {
	TotalRewardUnclaimed    *big.Int
	TotalSecondsClaimedX128 *big.Int
	NumberOfStakes          *big.Int
}, error) {
	return _Staker.Contract.Incentives(&_Staker.CallOpts, arg0)
}

// MaxIncentiveDuration is a free data retrieval call binding the contract method 0x3dc0714b.
//
// Solidity: function maxIncentiveDuration() view returns(uint256)
func (_Staker *StakerCaller) MaxIncentiveDuration(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "maxIncentiveDuration")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxIncentiveDuration is a free data retrieval call binding the contract method 0x3dc0714b.
//
// Solidity: function maxIncentiveDuration() view returns(uint256)
func (_Staker *StakerSession) MaxIncentiveDuration() (*big.Int, error) {
	return _Staker.Contract.MaxIncentiveDuration(&_Staker.CallOpts)
}

// MaxIncentiveDuration is a free data retrieval call binding the contract method 0x3dc0714b.
//
// Solidity: function maxIncentiveDuration() view returns(uint256)
func (_Staker *StakerCallerSession) MaxIncentiveDuration() (*big.Int, error) {
	return _Staker.Contract.MaxIncentiveDuration(&_Staker.CallOpts)
}

// MaxIncentiveStartLeadTime is a free data retrieval call binding the contract method 0x01b75440.
//
// Solidity: function maxIncentiveStartLeadTime() view returns(uint256)
func (_Staker *StakerCaller) MaxIncentiveStartLeadTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "maxIncentiveStartLeadTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxIncentiveStartLeadTime is a free data retrieval call binding the contract method 0x01b75440.
//
// Solidity: function maxIncentiveStartLeadTime() view returns(uint256)
func (_Staker *StakerSession) MaxIncentiveStartLeadTime() (*big.Int, error) {
	return _Staker.Contract.MaxIncentiveStartLeadTime(&_Staker.CallOpts)
}

// MaxIncentiveStartLeadTime is a free data retrieval call binding the contract method 0x01b75440.
//
// Solidity: function maxIncentiveStartLeadTime() view returns(uint256)
func (_Staker *StakerCallerSession) MaxIncentiveStartLeadTime() (*big.Int, error) {
	return _Staker.Contract.MaxIncentiveStartLeadTime(&_Staker.CallOpts)
}

// NonfungiblePositionManager is a free data retrieval call binding the contract method 0xb44a2722.
//
// Solidity: function nonfungiblePositionManager() view returns(address)
func (_Staker *StakerCaller) NonfungiblePositionManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "nonfungiblePositionManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NonfungiblePositionManager is a free data retrieval call binding the contract method 0xb44a2722.
//
// Solidity: function nonfungiblePositionManager() view returns(address)
func (_Staker *StakerSession) NonfungiblePositionManager() (common.Address, error) {
	return _Staker.Contract.NonfungiblePositionManager(&_Staker.CallOpts)
}

// NonfungiblePositionManager is a free data retrieval call binding the contract method 0xb44a2722.
//
// Solidity: function nonfungiblePositionManager() view returns(address)
func (_Staker *StakerCallerSession) NonfungiblePositionManager() (common.Address, error) {
	return _Staker.Contract.NonfungiblePositionManager(&_Staker.CallOpts)
}

// Rewards is a free data retrieval call binding the contract method 0xe70b9e27.
//
// Solidity: function rewards(address , address ) view returns(uint256)
func (_Staker *StakerCaller) Rewards(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "rewards", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Rewards is a free data retrieval call binding the contract method 0xe70b9e27.
//
// Solidity: function rewards(address , address ) view returns(uint256)
func (_Staker *StakerSession) Rewards(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _Staker.Contract.Rewards(&_Staker.CallOpts, arg0, arg1)
}

// Rewards is a free data retrieval call binding the contract method 0xe70b9e27.
//
// Solidity: function rewards(address , address ) view returns(uint256)
func (_Staker *StakerCallerSession) Rewards(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _Staker.Contract.Rewards(&_Staker.CallOpts, arg0, arg1)
}

// Stakes is a free data retrieval call binding the contract method 0xc36c1ea5.
//
// Solidity: function stakes(uint256 tokenId, bytes32 incentiveId) view returns(uint160 secondsPerLiquidityInsideInitialX128, uint128 liquidity)
func (_Staker *StakerCaller) Stakes(opts *bind.CallOpts, tokenId *big.Int, incentiveId [32]byte) (struct {
	SecondsPerLiquidityInsideInitialX128 *big.Int
	Liquidity                            *big.Int
}, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "stakes", tokenId, incentiveId)

	outstruct := new(struct {
		SecondsPerLiquidityInsideInitialX128 *big.Int
		Liquidity                            *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SecondsPerLiquidityInsideInitialX128 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Liquidity = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Stakes is a free data retrieval call binding the contract method 0xc36c1ea5.
//
// Solidity: function stakes(uint256 tokenId, bytes32 incentiveId) view returns(uint160 secondsPerLiquidityInsideInitialX128, uint128 liquidity)
func (_Staker *StakerSession) Stakes(tokenId *big.Int, incentiveId [32]byte) (struct {
	SecondsPerLiquidityInsideInitialX128 *big.Int
	Liquidity                            *big.Int
}, error) {
	return _Staker.Contract.Stakes(&_Staker.CallOpts, tokenId, incentiveId)
}

// Stakes is a free data retrieval call binding the contract method 0xc36c1ea5.
//
// Solidity: function stakes(uint256 tokenId, bytes32 incentiveId) view returns(uint160 secondsPerLiquidityInsideInitialX128, uint128 liquidity)
func (_Staker *StakerCallerSession) Stakes(tokenId *big.Int, incentiveId [32]byte) (struct {
	SecondsPerLiquidityInsideInitialX128 *big.Int
	Liquidity                            *big.Int
}, error) {
	return _Staker.Contract.Stakes(&_Staker.CallOpts, tokenId, incentiveId)
}

// ClaimReward is a paid mutator transaction binding the contract method 0x2f2d783d.
//
// Solidity: function claimReward(address rewardToken, address to, uint256 amountRequested) returns(uint256 reward)
func (_Staker *StakerTransactor) ClaimReward(opts *bind.TransactOpts, rewardToken common.Address, to common.Address, amountRequested *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "claimReward", rewardToken, to, amountRequested)
}

// ClaimReward is a paid mutator transaction binding the contract method 0x2f2d783d.
//
// Solidity: function claimReward(address rewardToken, address to, uint256 amountRequested) returns(uint256 reward)
func (_Staker *StakerSession) ClaimReward(rewardToken common.Address, to common.Address, amountRequested *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.ClaimReward(&_Staker.TransactOpts, rewardToken, to, amountRequested)
}

// ClaimReward is a paid mutator transaction binding the contract method 0x2f2d783d.
//
// Solidity: function claimReward(address rewardToken, address to, uint256 amountRequested) returns(uint256 reward)
func (_Staker *StakerTransactorSession) ClaimReward(rewardToken common.Address, to common.Address, amountRequested *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.ClaimReward(&_Staker.TransactOpts, rewardToken, to, amountRequested)
}

// CreateIncentive is a paid mutator transaction binding the contract method 0x5cc5e3d9.
//
// Solidity: function createIncentive((address,address,uint256,uint256,address) key, uint256 reward) returns()
func (_Staker *StakerTransactor) CreateIncentive(opts *bind.TransactOpts, key IUniswapV3StakerIncentiveKey, reward *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "createIncentive", key, reward)
}

// CreateIncentive is a paid mutator transaction binding the contract method 0x5cc5e3d9.
//
// Solidity: function createIncentive((address,address,uint256,uint256,address) key, uint256 reward) returns()
func (_Staker *StakerSession) CreateIncentive(key IUniswapV3StakerIncentiveKey, reward *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.CreateIncentive(&_Staker.TransactOpts, key, reward)
}

// CreateIncentive is a paid mutator transaction binding the contract method 0x5cc5e3d9.
//
// Solidity: function createIncentive((address,address,uint256,uint256,address) key, uint256 reward) returns()
func (_Staker *StakerTransactorSession) CreateIncentive(key IUniswapV3StakerIncentiveKey, reward *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.CreateIncentive(&_Staker.TransactOpts, key, reward)
}

// EndIncentive is a paid mutator transaction binding the contract method 0xb5ada6e4.
//
// Solidity: function endIncentive((address,address,uint256,uint256,address) key) returns(uint256 refund)
func (_Staker *StakerTransactor) EndIncentive(opts *bind.TransactOpts, key IUniswapV3StakerIncentiveKey) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "endIncentive", key)
}

// EndIncentive is a paid mutator transaction binding the contract method 0xb5ada6e4.
//
// Solidity: function endIncentive((address,address,uint256,uint256,address) key) returns(uint256 refund)
func (_Staker *StakerSession) EndIncentive(key IUniswapV3StakerIncentiveKey) (*types.Transaction, error) {
	return _Staker.Contract.EndIncentive(&_Staker.TransactOpts, key)
}

// EndIncentive is a paid mutator transaction binding the contract method 0xb5ada6e4.
//
// Solidity: function endIncentive((address,address,uint256,uint256,address) key) returns(uint256 refund)
func (_Staker *StakerTransactorSession) EndIncentive(key IUniswapV3StakerIncentiveKey) (*types.Transaction, error) {
	return _Staker.Contract.EndIncentive(&_Staker.TransactOpts, key)
}

// Multicall is a paid mutator transaction binding the contract method 0xac9650d8.
//
// Solidity: function multicall(bytes[] data) payable returns(bytes[] results)
func (_Staker *StakerTransactor) Multicall(opts *bind.TransactOpts, data [][]byte) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "multicall", data)
}

// Multicall is a paid mutator transaction binding the contract method 0xac9650d8.
//
// Solidity: function multicall(bytes[] data) payable returns(bytes[] results)
func (_Staker *StakerSession) Multicall(data [][]byte) (*types.Transaction, error) {
	return _Staker.Contract.Multicall(&_Staker.TransactOpts, data)
}

// Multicall is a paid mutator transaction binding the contract method 0xac9650d8.
//
// Solidity: function multicall(bytes[] data) payable returns(bytes[] results)
func (_Staker *StakerTransactorSession) Multicall(data [][]byte) (*types.Transaction, error) {
	return _Staker.Contract.Multicall(&_Staker.TransactOpts, data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address from, uint256 tokenId, bytes data) returns(bytes4)
func (_Staker *StakerTransactor) OnERC721Received(opts *bind.TransactOpts, arg0 common.Address, from common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "onERC721Received", arg0, from, tokenId, data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address from, uint256 tokenId, bytes data) returns(bytes4)
func (_Staker *StakerSession) OnERC721Received(arg0 common.Address, from common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Staker.Contract.OnERC721Received(&_Staker.TransactOpts, arg0, from, tokenId, data)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0x150b7a02.
//
// Solidity: function onERC721Received(address , address from, uint256 tokenId, bytes data) returns(bytes4)
func (_Staker *StakerTransactorSession) OnERC721Received(arg0 common.Address, from common.Address, tokenId *big.Int, data []byte) (*types.Transaction, error) {
	return _Staker.Contract.OnERC721Received(&_Staker.TransactOpts, arg0, from, tokenId, data)
}

// StakeToken is a paid mutator transaction binding the contract method 0xf2d2909b.
//
// Solidity: function stakeToken((address,address,uint256,uint256,address) key, uint256 tokenId) returns()
func (_Staker *StakerTransactor) StakeToken(opts *bind.TransactOpts, key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "stakeToken", key, tokenId)
}

// StakeToken is a paid mutator transaction binding the contract method 0xf2d2909b.
//
// Solidity: function stakeToken((address,address,uint256,uint256,address) key, uint256 tokenId) returns()
func (_Staker *StakerSession) StakeToken(key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.StakeToken(&_Staker.TransactOpts, key, tokenId)
}

// StakeToken is a paid mutator transaction binding the contract method 0xf2d2909b.
//
// Solidity: function stakeToken((address,address,uint256,uint256,address) key, uint256 tokenId) returns()
func (_Staker *StakerTransactorSession) StakeToken(key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.StakeToken(&_Staker.TransactOpts, key, tokenId)
}

// TransferDeposit is a paid mutator transaction binding the contract method 0x26bfee04.
//
// Solidity: function transferDeposit(uint256 tokenId, address to) returns()
func (_Staker *StakerTransactor) TransferDeposit(opts *bind.TransactOpts, tokenId *big.Int, to common.Address) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "transferDeposit", tokenId, to)
}

// TransferDeposit is a paid mutator transaction binding the contract method 0x26bfee04.
//
// Solidity: function transferDeposit(uint256 tokenId, address to) returns()
func (_Staker *StakerSession) TransferDeposit(tokenId *big.Int, to common.Address) (*types.Transaction, error) {
	return _Staker.Contract.TransferDeposit(&_Staker.TransactOpts, tokenId, to)
}

// TransferDeposit is a paid mutator transaction binding the contract method 0x26bfee04.
//
// Solidity: function transferDeposit(uint256 tokenId, address to) returns()
func (_Staker *StakerTransactorSession) TransferDeposit(tokenId *big.Int, to common.Address) (*types.Transaction, error) {
	return _Staker.Contract.TransferDeposit(&_Staker.TransactOpts, tokenId, to)
}

// UnstakeToken is a paid mutator transaction binding the contract method 0xf549ab42.
//
// Solidity: function unstakeToken((address,address,uint256,uint256,address) key, uint256 tokenId) returns()
func (_Staker *StakerTransactor) UnstakeToken(opts *bind.TransactOpts, key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "unstakeToken", key, tokenId)
}

// UnstakeToken is a paid mutator transaction binding the contract method 0xf549ab42.
//
// Solidity: function unstakeToken((address,address,uint256,uint256,address) key, uint256 tokenId) returns()
func (_Staker *StakerSession) UnstakeToken(key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.UnstakeToken(&_Staker.TransactOpts, key, tokenId)
}

// UnstakeToken is a paid mutator transaction binding the contract method 0xf549ab42.
//
// Solidity: function unstakeToken((address,address,uint256,uint256,address) key, uint256 tokenId) returns()
func (_Staker *StakerTransactorSession) UnstakeToken(key IUniswapV3StakerIncentiveKey, tokenId *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.UnstakeToken(&_Staker.TransactOpts, key, tokenId)
}

// WithdrawToken is a paid mutator transaction binding the contract method 0x3c423f0b.
//
// Solidity: function withdrawToken(uint256 tokenId, address to, bytes data) returns()
func (_Staker *StakerTransactor) WithdrawToken(opts *bind.TransactOpts, tokenId *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "withdrawToken", tokenId, to, data)
}

// WithdrawToken is a paid mutator transaction binding the contract method 0x3c423f0b.
//
// Solidity: function withdrawToken(uint256 tokenId, address to, bytes data) returns()
func (_Staker *StakerSession) WithdrawToken(tokenId *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Staker.Contract.WithdrawToken(&_Staker.TransactOpts, tokenId, to, data)
}

// WithdrawToken is a paid mutator transaction binding the contract method 0x3c423f0b.
//
// Solidity: function withdrawToken(uint256 tokenId, address to, bytes data) returns()
func (_Staker *StakerTransactorSession) WithdrawToken(tokenId *big.Int, to common.Address, data []byte) (*types.Transaction, error) {
	return _Staker.Contract.WithdrawToken(&_Staker.TransactOpts, tokenId, to, data)
}

// StakerDepositTransferredIterator is returned from FilterDepositTransferred and is used to iterate over the raw logs and unpacked data for DepositTransferred events raised by the Staker contract.
type StakerDepositTransferredIterator struct {
	Event *StakerDepositTransferred // Event containing the contract specifics and raw log

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
func (it *StakerDepositTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerDepositTransferred)
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
		it.Event = new(StakerDepositTransferred)
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
func (it *StakerDepositTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerDepositTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerDepositTransferred represents a DepositTransferred event raised by the Staker contract.
type StakerDepositTransferred struct {
	TokenId  *big.Int
	OldOwner common.Address
	NewOwner common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterDepositTransferred is a free log retrieval operation binding the contract event 0xcdfc765b85e1048bee3c6a0f9d1c91fc7c4631f5fe5745a55fc6843db5c3260f.
//
// Solidity: event DepositTransferred(uint256 indexed tokenId, address indexed oldOwner, address indexed newOwner)
func (_Staker *StakerFilterer) FilterDepositTransferred(opts *bind.FilterOpts, tokenId []*big.Int, oldOwner []common.Address, newOwner []common.Address) (*StakerDepositTransferredIterator, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "DepositTransferred", tokenIdRule, oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &StakerDepositTransferredIterator{contract: _Staker.contract, event: "DepositTransferred", logs: logs, sub: sub}, nil
}

// WatchDepositTransferred is a free log subscription operation binding the contract event 0xcdfc765b85e1048bee3c6a0f9d1c91fc7c4631f5fe5745a55fc6843db5c3260f.
//
// Solidity: event DepositTransferred(uint256 indexed tokenId, address indexed oldOwner, address indexed newOwner)
func (_Staker *StakerFilterer) WatchDepositTransferred(opts *bind.WatchOpts, sink chan<- *StakerDepositTransferred, tokenId []*big.Int, oldOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var oldOwnerRule []interface{}
	for _, oldOwnerItem := range oldOwner {
		oldOwnerRule = append(oldOwnerRule, oldOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "DepositTransferred", tokenIdRule, oldOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerDepositTransferred)
				if err := _Staker.contract.UnpackLog(event, "DepositTransferred", log); err != nil {
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

// ParseDepositTransferred is a log parse operation binding the contract event 0xcdfc765b85e1048bee3c6a0f9d1c91fc7c4631f5fe5745a55fc6843db5c3260f.
//
// Solidity: event DepositTransferred(uint256 indexed tokenId, address indexed oldOwner, address indexed newOwner)
func (_Staker *StakerFilterer) ParseDepositTransferred(log types.Log) (*StakerDepositTransferred, error) {
	event := new(StakerDepositTransferred)
	if err := _Staker.contract.UnpackLog(event, "DepositTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerIncentiveCreatedIterator is returned from FilterIncentiveCreated and is used to iterate over the raw logs and unpacked data for IncentiveCreated events raised by the Staker contract.
type StakerIncentiveCreatedIterator struct {
	Event *StakerIncentiveCreated // Event containing the contract specifics and raw log

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
func (it *StakerIncentiveCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerIncentiveCreated)
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
		it.Event = new(StakerIncentiveCreated)
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
func (it *StakerIncentiveCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerIncentiveCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerIncentiveCreated represents a IncentiveCreated event raised by the Staker contract.
type StakerIncentiveCreated struct {
	RewardToken common.Address
	Pool        common.Address
	StartTime   *big.Int
	EndTime     *big.Int
	Refundee    common.Address
	Reward      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterIncentiveCreated is a free log retrieval operation binding the contract event 0xa876344e28d4b5191ad03bc0d43f740e3695827ab0faccac739930b915ef8b02.
//
// Solidity: event IncentiveCreated(address indexed rewardToken, address indexed pool, uint256 startTime, uint256 endTime, address refundee, uint256 reward)
func (_Staker *StakerFilterer) FilterIncentiveCreated(opts *bind.FilterOpts, rewardToken []common.Address, pool []common.Address) (*StakerIncentiveCreatedIterator, error) {

	var rewardTokenRule []interface{}
	for _, rewardTokenItem := range rewardToken {
		rewardTokenRule = append(rewardTokenRule, rewardTokenItem)
	}
	var poolRule []interface{}
	for _, poolItem := range pool {
		poolRule = append(poolRule, poolItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "IncentiveCreated", rewardTokenRule, poolRule)
	if err != nil {
		return nil, err
	}
	return &StakerIncentiveCreatedIterator{contract: _Staker.contract, event: "IncentiveCreated", logs: logs, sub: sub}, nil
}

// WatchIncentiveCreated is a free log subscription operation binding the contract event 0xa876344e28d4b5191ad03bc0d43f740e3695827ab0faccac739930b915ef8b02.
//
// Solidity: event IncentiveCreated(address indexed rewardToken, address indexed pool, uint256 startTime, uint256 endTime, address refundee, uint256 reward)
func (_Staker *StakerFilterer) WatchIncentiveCreated(opts *bind.WatchOpts, sink chan<- *StakerIncentiveCreated, rewardToken []common.Address, pool []common.Address) (event.Subscription, error) {

	var rewardTokenRule []interface{}
	for _, rewardTokenItem := range rewardToken {
		rewardTokenRule = append(rewardTokenRule, rewardTokenItem)
	}
	var poolRule []interface{}
	for _, poolItem := range pool {
		poolRule = append(poolRule, poolItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "IncentiveCreated", rewardTokenRule, poolRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerIncentiveCreated)
				if err := _Staker.contract.UnpackLog(event, "IncentiveCreated", log); err != nil {
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

// ParseIncentiveCreated is a log parse operation binding the contract event 0xa876344e28d4b5191ad03bc0d43f740e3695827ab0faccac739930b915ef8b02.
//
// Solidity: event IncentiveCreated(address indexed rewardToken, address indexed pool, uint256 startTime, uint256 endTime, address refundee, uint256 reward)
func (_Staker *StakerFilterer) ParseIncentiveCreated(log types.Log) (*StakerIncentiveCreated, error) {
	event := new(StakerIncentiveCreated)
	if err := _Staker.contract.UnpackLog(event, "IncentiveCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerIncentiveEndedIterator is returned from FilterIncentiveEnded and is used to iterate over the raw logs and unpacked data for IncentiveEnded events raised by the Staker contract.
type StakerIncentiveEndedIterator struct {
	Event *StakerIncentiveEnded // Event containing the contract specifics and raw log

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
func (it *StakerIncentiveEndedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerIncentiveEnded)
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
		it.Event = new(StakerIncentiveEnded)
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
func (it *StakerIncentiveEndedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerIncentiveEndedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerIncentiveEnded represents a IncentiveEnded event raised by the Staker contract.
type StakerIncentiveEnded struct {
	IncentiveId [32]byte
	Refund      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterIncentiveEnded is a free log retrieval operation binding the contract event 0x65124e6175aa9904f40735e87e2a37c76e87a609b855287bb4d1aba8257d9763.
//
// Solidity: event IncentiveEnded(bytes32 indexed incentiveId, uint256 refund)
func (_Staker *StakerFilterer) FilterIncentiveEnded(opts *bind.FilterOpts, incentiveId [][32]byte) (*StakerIncentiveEndedIterator, error) {

	var incentiveIdRule []interface{}
	for _, incentiveIdItem := range incentiveId {
		incentiveIdRule = append(incentiveIdRule, incentiveIdItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "IncentiveEnded", incentiveIdRule)
	if err != nil {
		return nil, err
	}
	return &StakerIncentiveEndedIterator{contract: _Staker.contract, event: "IncentiveEnded", logs: logs, sub: sub}, nil
}

// WatchIncentiveEnded is a free log subscription operation binding the contract event 0x65124e6175aa9904f40735e87e2a37c76e87a609b855287bb4d1aba8257d9763.
//
// Solidity: event IncentiveEnded(bytes32 indexed incentiveId, uint256 refund)
func (_Staker *StakerFilterer) WatchIncentiveEnded(opts *bind.WatchOpts, sink chan<- *StakerIncentiveEnded, incentiveId [][32]byte) (event.Subscription, error) {

	var incentiveIdRule []interface{}
	for _, incentiveIdItem := range incentiveId {
		incentiveIdRule = append(incentiveIdRule, incentiveIdItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "IncentiveEnded", incentiveIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerIncentiveEnded)
				if err := _Staker.contract.UnpackLog(event, "IncentiveEnded", log); err != nil {
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

// ParseIncentiveEnded is a log parse operation binding the contract event 0x65124e6175aa9904f40735e87e2a37c76e87a609b855287bb4d1aba8257d9763.
//
// Solidity: event IncentiveEnded(bytes32 indexed incentiveId, uint256 refund)
func (_Staker *StakerFilterer) ParseIncentiveEnded(log types.Log) (*StakerIncentiveEnded, error) {
	event := new(StakerIncentiveEnded)
	if err := _Staker.contract.UnpackLog(event, "IncentiveEnded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerRewardClaimedIterator is returned from FilterRewardClaimed and is used to iterate over the raw logs and unpacked data for RewardClaimed events raised by the Staker contract.
type StakerRewardClaimedIterator struct {
	Event *StakerRewardClaimed // Event containing the contract specifics and raw log

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
func (it *StakerRewardClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerRewardClaimed)
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
		it.Event = new(StakerRewardClaimed)
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
func (it *StakerRewardClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerRewardClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerRewardClaimed represents a RewardClaimed event raised by the Staker contract.
type StakerRewardClaimed struct {
	To     common.Address
	Reward *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterRewardClaimed is a free log retrieval operation binding the contract event 0x106f923f993c2149d49b4255ff723acafa1f2d94393f561d3eda32ae348f7241.
//
// Solidity: event RewardClaimed(address indexed to, uint256 reward)
func (_Staker *StakerFilterer) FilterRewardClaimed(opts *bind.FilterOpts, to []common.Address) (*StakerRewardClaimedIterator, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "RewardClaimed", toRule)
	if err != nil {
		return nil, err
	}
	return &StakerRewardClaimedIterator{contract: _Staker.contract, event: "RewardClaimed", logs: logs, sub: sub}, nil
}

// WatchRewardClaimed is a free log subscription operation binding the contract event 0x106f923f993c2149d49b4255ff723acafa1f2d94393f561d3eda32ae348f7241.
//
// Solidity: event RewardClaimed(address indexed to, uint256 reward)
func (_Staker *StakerFilterer) WatchRewardClaimed(opts *bind.WatchOpts, sink chan<- *StakerRewardClaimed, to []common.Address) (event.Subscription, error) {

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "RewardClaimed", toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerRewardClaimed)
				if err := _Staker.contract.UnpackLog(event, "RewardClaimed", log); err != nil {
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

// ParseRewardClaimed is a log parse operation binding the contract event 0x106f923f993c2149d49b4255ff723acafa1f2d94393f561d3eda32ae348f7241.
//
// Solidity: event RewardClaimed(address indexed to, uint256 reward)
func (_Staker *StakerFilterer) ParseRewardClaimed(log types.Log) (*StakerRewardClaimed, error) {
	event := new(StakerRewardClaimed)
	if err := _Staker.contract.UnpackLog(event, "RewardClaimed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerTokenStakedIterator is returned from FilterTokenStaked and is used to iterate over the raw logs and unpacked data for TokenStaked events raised by the Staker contract.
type StakerTokenStakedIterator struct {
	Event *StakerTokenStaked // Event containing the contract specifics and raw log

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
func (it *StakerTokenStakedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerTokenStaked)
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
		it.Event = new(StakerTokenStaked)
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
func (it *StakerTokenStakedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerTokenStakedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerTokenStaked represents a TokenStaked event raised by the Staker contract.
type StakerTokenStaked struct {
	TokenId     *big.Int
	IncentiveId [32]byte
	Liquidity   *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTokenStaked is a free log retrieval operation binding the contract event 0x3fe90ccd0a34e28f2b4b7a1e8323415ed9dd595f4eec5dfd461d18c2df336dbd.
//
// Solidity: event TokenStaked(uint256 indexed tokenId, bytes32 indexed incentiveId, uint128 liquidity)
func (_Staker *StakerFilterer) FilterTokenStaked(opts *bind.FilterOpts, tokenId []*big.Int, incentiveId [][32]byte) (*StakerTokenStakedIterator, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var incentiveIdRule []interface{}
	for _, incentiveIdItem := range incentiveId {
		incentiveIdRule = append(incentiveIdRule, incentiveIdItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "TokenStaked", tokenIdRule, incentiveIdRule)
	if err != nil {
		return nil, err
	}
	return &StakerTokenStakedIterator{contract: _Staker.contract, event: "TokenStaked", logs: logs, sub: sub}, nil
}

// WatchTokenStaked is a free log subscription operation binding the contract event 0x3fe90ccd0a34e28f2b4b7a1e8323415ed9dd595f4eec5dfd461d18c2df336dbd.
//
// Solidity: event TokenStaked(uint256 indexed tokenId, bytes32 indexed incentiveId, uint128 liquidity)
func (_Staker *StakerFilterer) WatchTokenStaked(opts *bind.WatchOpts, sink chan<- *StakerTokenStaked, tokenId []*big.Int, incentiveId [][32]byte) (event.Subscription, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var incentiveIdRule []interface{}
	for _, incentiveIdItem := range incentiveId {
		incentiveIdRule = append(incentiveIdRule, incentiveIdItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "TokenStaked", tokenIdRule, incentiveIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerTokenStaked)
				if err := _Staker.contract.UnpackLog(event, "TokenStaked", log); err != nil {
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

// ParseTokenStaked is a log parse operation binding the contract event 0x3fe90ccd0a34e28f2b4b7a1e8323415ed9dd595f4eec5dfd461d18c2df336dbd.
//
// Solidity: event TokenStaked(uint256 indexed tokenId, bytes32 indexed incentiveId, uint128 liquidity)
func (_Staker *StakerFilterer) ParseTokenStaked(log types.Log) (*StakerTokenStaked, error) {
	event := new(StakerTokenStaked)
	if err := _Staker.contract.UnpackLog(event, "TokenStaked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerTokenUnstakedIterator is returned from FilterTokenUnstaked and is used to iterate over the raw logs and unpacked data for TokenUnstaked events raised by the Staker contract.
type StakerTokenUnstakedIterator struct {
	Event *StakerTokenUnstaked // Event containing the contract specifics and raw log

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
func (it *StakerTokenUnstakedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerTokenUnstaked)
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
		it.Event = new(StakerTokenUnstaked)
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
func (it *StakerTokenUnstakedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerTokenUnstakedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerTokenUnstaked represents a TokenUnstaked event raised by the Staker contract.
type StakerTokenUnstaked struct {
	TokenId     *big.Int
	IncentiveId [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTokenUnstaked is a free log retrieval operation binding the contract event 0xe1ba67e807ae0efa0a9549f9520ddc15c27f0a4dae2bc045e800ca66a940778f.
//
// Solidity: event TokenUnstaked(uint256 indexed tokenId, bytes32 indexed incentiveId)
func (_Staker *StakerFilterer) FilterTokenUnstaked(opts *bind.FilterOpts, tokenId []*big.Int, incentiveId [][32]byte) (*StakerTokenUnstakedIterator, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var incentiveIdRule []interface{}
	for _, incentiveIdItem := range incentiveId {
		incentiveIdRule = append(incentiveIdRule, incentiveIdItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "TokenUnstaked", tokenIdRule, incentiveIdRule)
	if err != nil {
		return nil, err
	}
	return &StakerTokenUnstakedIterator{contract: _Staker.contract, event: "TokenUnstaked", logs: logs, sub: sub}, nil
}

// WatchTokenUnstaked is a free log subscription operation binding the contract event 0xe1ba67e807ae0efa0a9549f9520ddc15c27f0a4dae2bc045e800ca66a940778f.
//
// Solidity: event TokenUnstaked(uint256 indexed tokenId, bytes32 indexed incentiveId)
func (_Staker *StakerFilterer) WatchTokenUnstaked(opts *bind.WatchOpts, sink chan<- *StakerTokenUnstaked, tokenId []*big.Int, incentiveId [][32]byte) (event.Subscription, error) {

	var tokenIdRule []interface{}
	for _, tokenIdItem := range tokenId {
		tokenIdRule = append(tokenIdRule, tokenIdItem)
	}
	var incentiveIdRule []interface{}
	for _, incentiveIdItem := range incentiveId {
		incentiveIdRule = append(incentiveIdRule, incentiveIdItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "TokenUnstaked", tokenIdRule, incentiveIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerTokenUnstaked)
				if err := _Staker.contract.UnpackLog(event, "TokenUnstaked", log); err != nil {
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

// ParseTokenUnstaked is a log parse operation binding the contract event 0xe1ba67e807ae0efa0a9549f9520ddc15c27f0a4dae2bc045e800ca66a940778f.
//
// Solidity: event TokenUnstaked(uint256 indexed tokenId, bytes32 indexed incentiveId)
func (_Staker *StakerFilterer) ParseTokenUnstaked(log types.Log) (*StakerTokenUnstaked, error) {
	event := new(StakerTokenUnstaked)
	if err := _Staker.contract.UnpackLog(event, "TokenUnstaked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
