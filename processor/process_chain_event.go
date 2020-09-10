package processor

import (
	"errors"
)

var (
	ErrNotAnERC20Event                                = errors.New("not an erc20 event")
	ErrNotABuiltinAssetEvent                          = errors.New("not an builtin asset event")
	ErrUnsupportedEventAction                         = errors.New("unsupported event action")
	ErrChainEventAssetListERC20WithoutEnoughSignature = errors.New("chain event for erc20 asset list received with missing node signatures")
)
