package account

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// MinPaymasterAndDataLen represents the minimum length of the PaymasterAndData field in a PackedUserOperation,
// including paymaster address (20), validationGas (16) and postOpGas (16).
const MinPaymasterAndDataLen = 52

// Factory extracts the factory address from the InitCode field of the user operation.
func (userOp *PackedUserOperation) Factory() common.Address {
	if len(userOp.InitCode) < 20 {
		return common.Address{}
	}

	return common.BytesToAddress(userOp.InitCode[:20])
}

// FactoryData extracts the factory initialization data from the InitCode field of the user operation.
func (userOp *PackedUserOperation) FactoryData() []byte {
	if len(userOp.InitCode) < 20 {
		return nil
	}

	return userOp.InitCode[20:]
}

// VerificationGasLimit extracts the verification gas limit from the AccountGasLimits field of the user operation.
func (userOp *PackedUserOperation) VerificationGasLimit() *big.Int {
	return new(big.Int).SetBytes(userOp.AccountGasLimits[:16])
}

// CallGasLimit extracts the call gas limit from the AccountGasLimits field of the user operation.
func (userOp *PackedUserOperation) CallGasLimit() *big.Int {
	return new(big.Int).SetBytes(userOp.AccountGasLimits[16:])
}

// SetAccountGasLimits sets the verification and call gas limits in the AccountGasLimits field of the user operation.
func (userOp *PackedUserOperation) SetAccountGasLimits(verificationGasLimit, callGasLimit *big.Int) {
	SafeBigFillBytes(verificationGasLimit, userOp.AccountGasLimits[:16])
	SafeBigFillBytes(callGasLimit, userOp.AccountGasLimits[16:])
}

// MaxPriorityFeePerGas extracts the max priority fee per gas from the GasFees field of the user operation.
func (userOp *PackedUserOperation) MaxPriorityFeePerGas() *big.Int {
	return new(big.Int).SetBytes(userOp.GasFees[:16])
}

// MaxFeePerGas extracts the max fee per gas from the GasFees field of the user operation.
func (userOp *PackedUserOperation) MaxFeePerGas() *big.Int {
	return new(big.Int).SetBytes(userOp.GasFees[16:])
}

// SetGasFees sets the max priority fee per gas and max fee per gas in the GasFees field of the user operation.
func (userOp *PackedUserOperation) SetGasFees(maxPriorityFeePerGas, maxFeePerGas *big.Int) {
	SafeBigFillBytes(maxPriorityFeePerGas, userOp.GasFees[:16])
	SafeBigFillBytes(maxFeePerGas, userOp.GasFees[16:])
}

// Paymaster extracts the paymaster address from the PaymasterAndData field of the user operation.
func (userOp *PackedUserOperation) Paymaster() common.Address {
	if len(userOp.PaymasterAndData) < MinPaymasterAndDataLen {
		return common.Address{}
	}

	return common.BytesToAddress(userOp.PaymasterAndData[:20])
}

// PaymasterVerificationGasLimit extracts the paymaster verification gas limit from the PaymasterAndData field of the user operation.
func (userOp *PackedUserOperation) PaymasterVerificationGasLimit() *big.Int {
	if len(userOp.PaymasterAndData) < MinPaymasterAndDataLen {
		return big.NewInt(0)
	}

	return new(big.Int).SetBytes(userOp.PaymasterAndData[20:36])
}

// PaymasterPostOpGasLimit extracts the paymaster post-operation gas limit from the PaymasterAndData field of the user operation.
func (userOp *PackedUserOperation) PaymasterPostOpGasLimit() *big.Int {
	if len(userOp.PaymasterAndData) < MinPaymasterAndDataLen {
		return big.NewInt(0)
	}

	return new(big.Int).SetBytes(userOp.PaymasterAndData[36:52])
}

// SetPaymasterGasLimits sets the paymaster verification and post-operation gas limits in the PaymasterAndData field of the user operation.
func (userOp *PackedUserOperation) SetPaymasterGasLimits(paymasterVerificationGasLimit, paymasterPostOpGasLimit *big.Int) {
	if len(userOp.PaymasterAndData) >= MinPaymasterAndDataLen {
		SafeBigFillBytes(paymasterVerificationGasLimit, userOp.PaymasterAndData[20:36])
		SafeBigFillBytes(paymasterPostOpGasLimit, userOp.PaymasterAndData[36:52])
	}
}

// PaymasterData extracts the custom paymaster data from the PaymasterAndData field of the user operation.
func (userOp *PackedUserOperation) PaymasterData() []byte {
	if len(userOp.PaymasterAndData) < MinPaymasterAndDataLen {
		return nil
	}

	return userOp.PaymasterAndData[MinPaymasterAndDataLen:]
}

// SetPaymasterAndData sets the paymaster address, gas limits, and custom paymaster data in the PaymasterAndData field of the user operation.
func (userOp *PackedUserOperation) SetPaymasterAndData(paymaster common.Address, paymasterVerificationGasLimit, paymasterPostOpGasLimit *big.Int, paymasterData []byte) {
	userOp.PaymasterAndData = make([]byte, MinPaymasterAndDataLen+len(paymasterData))

	copy(userOp.PaymasterAndData[:20], paymaster.Bytes())
	SafeBigFillBytes(paymasterVerificationGasLimit, userOp.PaymasterAndData[20:36])
	SafeBigFillBytes(paymasterPostOpGasLimit, userOp.PaymasterAndData[36:52])
	copy(userOp.PaymasterAndData[52:], paymasterData)
}

// MaxGasCost calculates the maximum gas cost of the user operation, including pre-verification gas, verification gas, call gas,
// and paymaster gas limits, multiplied by the max fee per gas.
func (userOp *PackedUserOperation) MaxGasCost() *big.Int {
	maxCost := big.NewInt(0)

	maxCost.Add(maxCost, userOp.PreVerificationGas)
	maxCost.Add(maxCost, userOp.VerificationGasLimit())
	maxCost.Add(maxCost, userOp.CallGasLimit())
	maxCost.Add(maxCost, userOp.PaymasterVerificationGasLimit())
	maxCost.Add(maxCost, userOp.PaymasterPostOpGasLimit())

	maxCost.Mul(maxCost, userOp.MaxFeePerGas())

	return maxCost
}
