package governance

import (
	"code.vegaprotocol.io/vega/types"
	"github.com/pkg/errors"
)

var (
	ErrFreeformDescriptionTooLong = errors.New("freeform description too long")
	ErrFreeformParameterEmpty     = errors.New("freeform parameter empty")
)

func validateNewFreeform(f *types.NewFreeform) (types.ProposalError, error) {
	if len(f.Changes.URL) == 0 || len(f.Changes.Hash) == 0 {
		return types.ProposalErrorInvalidFreeform, ErrFreeformParameterEmpty
	}

	if len(f.Changes.Description) > 255 {
		return types.ProposalErrorInvalidFreeform, ErrFreeformDescriptionTooLong
	}
	return types.ProposalErrorUnspecified, nil
}
