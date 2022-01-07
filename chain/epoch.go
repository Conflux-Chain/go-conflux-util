package chain

import (
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/pkg/errors"
)

// ConvertToNumberedEpoch converts named epoch to numbered epoch if necessary.
func ConvertToNumberedEpoch(cfx sdk.ClientOperator, epoch *types.Epoch) (*types.Epoch, error) {
	if _, ok := epoch.ToInt(); ok { // already a numbered epoch
		return epoch, nil
	}

	epochNum, err := cfx.GetEpochNumber(epoch)
	if err != nil {
		return nil, errors.WithMessagef(
			err, "failed to get epoch number for named epoch %v", epoch,
		)
	}

	return types.NewEpochNumber(epochNum), nil
}
