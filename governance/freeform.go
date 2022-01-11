package governance

import (
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"github.com/pkg/errors"
)

var (
	ErrFreeformDescriptionTooLong = errors.New("freeform description too long")
	ErrFreeformParameterEmpty     = errors.New("freeform parameter empty")
)

func validateNewFreeform(f *types.NewFreeform) (types.ProposalError, error) {
	if len(f.URL) == 0 || len(f.Hash) == 0 {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_FREEFORM, ErrFreeformParameterEmpty
	}

	if len(f.Description) > 255 {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_FREEFORM, ErrFreeformDescriptionTooLong
	}
	return proto.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
