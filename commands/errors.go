package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrIsRequired                                      = errors.New("is required")
	ErrMustBePositive                                  = errors.New("must be positive")
	ErrMustBePositiveOrZero                            = errors.New("must be positive or zero")
	ErrMustBeNegativeOrZero                            = errors.New("must be negative or zero")
	ErrMustBeLessThan150                               = errors.New("must be less than 150")
	ErrMustBeAtMost1M                                  = errors.New("must be at most 1000000")
	ErrMustBeAtMost100                                 = errors.New("must be at most 100")
	ErrMustBeWithinRange7                              = errors.New("must be between -7 and 7")
	ErrIsNotValid                                      = errors.New("is not a valid value")
	ErrIsNotValidWithOCO                               = errors.New("is not a valid with one cancel other")
	ErrIsNotValidNumber                                = errors.New("is not a valid number")
	ErrIsNotSupported                                  = errors.New("is not supported")
	ErrIsUnauthorised                                  = errors.New("is unauthorised")
	ErrCannotAmendToGFA                                = errors.New("cannot amend to time in force GFA")
	ErrCannotAmendToGFN                                = errors.New("cannot amend to time in force GFN")
	ErrNonGTTOrderWithExpiry                           = errors.New("non GTT order with expiry")
	ErrGTTOrderWithNoExpiry                            = errors.New("GTT order without expiry")
	ErrIsMismatching                                   = errors.New("is mismatching")
	ErrReferenceTooLong                                = errors.New("reference is too long")
	ErrNotAValidInteger                                = errors.New("not a valid integer")
	ErrNotAValidFloat                                  = errors.New("not a valid float")
	ErrMustBeLessThan100Chars                          = errors.New("must be less than 100 characters")
	ErrMustNotExceed20000Chars                         = errors.New("must not exceed 20000 characters")
	ErrShouldBeHexEncoded                              = errors.New("should be hex encoded")
	ErrSignatureNotVerifiable                          = errors.New("signature is not verifiable")
	ErrInvalidSignature                                = errors.New("invalid signature")
	ErrUnsupportedAlgorithm                            = errors.New("unsupported algorithm")
	ErrEmptyBatchMarketInstructions                    = errors.New("empty batch market instructions")
	ErrIsNotValidVegaPubkey                            = errors.New("is not a valid vega public key")
	ErrIsNotValidEthereumAddress                       = errors.New("is not a valid ethereum address")
	ErrEmptyEthereumCallSpec                           = errors.New("ethereum call spec is required")
	ErrInvalidEthereumAbi                              = errors.New("is not a valid ethereum abi definition")
	ErrInvalidEthereumCallTrigger                      = errors.New("ethereum call trigger not valid")
	ErrInvalidEthereumCallArgs                         = errors.New("ethereum call arguments not valid")
	ErrInvalidEthereumFilters                          = errors.New("ethereum call filters not valid")
	ErrInvalidEthereumCallSpec                         = errors.New("ethereum call spec is not valid")
	ErrMustBeWithinRange01                             = errors.New("must be between 0 and 1")
	ErrMustBeWithinRange11                             = errors.New("must be between -1 and 1")
	ErrMustBeLTE1                                      = errors.New("must be less than or equal to 1")
	ErrMustBeReduceOnly                                = errors.New("must be reduce only")
	ErrExpiryStrategyRequiredWhenExpiresAtSet          = errors.New("expiry strategy required when expires_at set")
	ErrMustHaveAtLeastOneOfRisesAboveOrFallsBelow      = errors.New("must have at least one of rises above or falls below")
	ErrMustHaveAStopOrderTrigger                       = errors.New("must have a stop order trigger")
	ErrFallsBelowAndRiseAboveMarketIDMustBeTheSame     = errors.New("market ID for falls below and rises above must be the same")
	ErrTrailingPercentOffsetMinimalIncrementNotReached = errors.New("trailing percent offset minimal increment must be >= 0.001")
	ErrMustBeEmpty                                     = errors.New("must be empty")
	ErrMustBeGTEClampLowerBound                        = errors.New("must be greater than or equal to clamp lower bound")
	ErrOneTimeTriggerAllowedMax                        = errors.New("maximum one time trigger allowed")
	ErrMustBeGreaterThanEnactmentTimestamp             = errors.New("must be greater than proposal_submission.terms.enactment_timestamp")
	ErrMustBeWithinRange1366                           = errors.New("must be between 1 and 366")
)

type Errors map[string][]error

func NewErrors() Errors {
	return Errors{}
}

func (e Errors) Error() string {
	if len(e) <= 0 {
		return ""
	}

	propMessages := []string{}
	for prop, errs := range e {
		errMessages := make([]string, 0, len(errs))
		for _, err := range errs {
			errMessages = append(errMessages, err.Error())
		}
		propMessageFmt := fmt.Sprintf("%v (%v)", prop, strings.Join(errMessages, ", "))
		propMessages = append(propMessages, propMessageFmt)
	}

	sort.Strings(propMessages)
	return strings.Join(propMessages, ", ")
}

func (e Errors) Empty() bool {
	return len(e) == 0
}

// AddForProperty adds an error for a given property.
func (e Errors) AddForProperty(prop string, err error) {
	errs, ok := e[prop]
	if !ok {
		errs = []error{}
	}

	e[prop] = append(errs, err)
}

// FinalAddForProperty behaves like AddForProperty, but is meant to be called in
// a "return" statement. This helper is usually used for terminal errors.
func (e Errors) FinalAddForProperty(prop string, err error) Errors {
	e.AddForProperty(prop, err)
	return e
}

// Add adds a general error that is not related to a specific property.
func (e Errors) Add(err error) {
	e.AddForProperty("*", err)
}

// FinalAdd behaves like Add, but is meant to be called in a "return" statement.
// This helper is usually used for terminal errors.
func (e Errors) FinalAdd(err error) Errors {
	e.Add(err)
	return e
}

func (e Errors) Merge(oth Errors) {
	if oth == nil {
		return
	}

	for prop, errs := range oth {
		for _, err := range errs {
			e.AddForProperty(prop, err)
		}
	}
}

func (e Errors) Get(prop string) []error {
	messages, ok := e[prop]
	if !ok {
		return nil
	}
	return messages
}

func (e Errors) ErrorOrNil() error {
	if len(e) <= 0 {
		return nil
	}
	return e
}

func (e Errors) MarshalJSON() ([]byte, error) {
	out := map[string][]string{}
	for prop, errs := range e {
		messages := []string{}
		for _, err := range errs {
			messages = append(messages, err.Error())
		}
		out[prop] = messages
	}
	return json.Marshal(out)
}
