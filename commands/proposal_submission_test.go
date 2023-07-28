package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/libs/test"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmission(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilProposalSubmissionFails)
	t.Run("Submitting a proposal change without change fails", testProposalSubmissionWithoutChangeFails)
	t.Run("Submitting a proposal without terms fails", testProposalSubmissionWithoutTermsFails)
	t.Run("Submitting a proposal with non-positive closing timestamp fails", testProposalSubmissionWithNonPositiveClosingTimestampFails)
	t.Run("Submitting a proposal with positive closing timestamp succeeds", testProposalSubmissionWithPositiveClosingTimestampSucceeds)
	t.Run("Submitting a proposal with non-positive enactment timestamp fails", testProposalSubmissionWithNonPositiveEnactmentTimestampFails)
	t.Run("Submitting a proposal with positive enactment timestamp succeeds", testProposalSubmissionWithPositiveEnactmentTimestampSucceeds)
	t.Run("Submitting a proposal with negative validation timestamp fails", testProposalSubmissionWithNegativeValidationTimestampFails)
	t.Run("Submitting a proposal with positive validation timestamp succeeds", testProposalSubmissionWithPositiveValidationTimestampSucceeds)
	t.Run("Submitting a proposal with closing timestamp after enactment timestamp fails", testProposalSubmissionWithClosingTimestampAfterEnactmentTimestampFails)
	t.Run("Submitting a proposal with closing timestamp before enactment timestamp succeeds", testProposalSubmissionWithClosingTimestampBeforeEnactmentTimestampSucceeds)
	t.Run("Submitting a proposal with closing timestamp at enactment timestamp succeeds", testProposalSubmissionWithClosingTimestampAtEnactmentTimestampSucceeds)
	t.Run("Submitting a proposal with validation timestamp after closing timestamp fails", testProposalSubmissionWithValidationTimestampAfterClosingTimestampFails)
	t.Run("Submitting a proposal with validation timestamp at closing timestamp succeeds", testProposalSubmissionWithValidationTimestampAtClosingTimestampFails)
	t.Run("Submitting a proposal with validation timestamp before closing timestamp fails", testProposalSubmissionWithValidationTimestampBeforeClosingTimestampSucceeds)
	t.Run("Submitting a proposal without rational fails", testProposalSubmissionWithoutRationalFails)
	t.Run("Submitting a proposal with rational succeeds", testProposalSubmissionWithRationalSucceeds)
	t.Run("Submitting a proposal with rational description succeeds", testProposalSubmissionWithRationalDescriptionSucceeds)
	t.Run("Submitting a proposal with incorrect rational description fails", testProposalSubmissionWithIncorrectRationalDescriptionFails)
	t.Run("Submitting a proposal with rational URL and hash succeeds", testProposalSubmissionWithRationalDescriptionAndTitleSucceeds)
}

func testNilProposalSubmissionFails(t *testing.T) {
	err := checkProposalSubmission(nil)

	assert.Contains(t, err.Get("proposal_submission"), commands.ErrIsRequired)
}

func testProposalSubmissionWithoutTermsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{})

	assert.Contains(t, err.Get("proposal_submission.terms"), commands.ErrIsRequired)
}

func testProposalSubmissionWithoutChangeFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change"), commands.ErrIsRequired)
}

func testProposalSubmissionWithNonPositiveClosingTimestampFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value int64
	}{
		{
			msg:   "with 0 as closing timestamp",
			value: 0,
		}, {
			msg:   "with negative closing timestamp",
			value: test.RandomNegativeI64(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					ClosingTimestamp: tc.value,
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.closing_timestamp"), commands.ErrMustBePositive)
		})
	}
}

func testProposalSubmissionWithPositiveClosingTimestampSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp: test.RandomPositiveI64(),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.closing_timestamp"), commands.ErrMustBePositive)
}

func testProposalSubmissionWithNonPositiveEnactmentTimestampFails(t *testing.T) {
	testCases := []struct {
		msg   string
		value int64
	}{
		{
			msg:   "with 0 as closing timestamp",
			value: 0,
		}, {
			msg:   "with negative closing timestamp",
			value: test.RandomNegativeI64(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					EnactmentTimestamp: tc.value,
				},
			})

			assert.Contains(t, err.Get("proposal_submission.terms.enactment_timestamp"), commands.ErrMustBePositive)
		})
	}
}

func testProposalSubmissionWithPositiveEnactmentTimestampSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			EnactmentTimestamp: test.RandomPositiveI64(),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.enactment_timestamp"), commands.ErrMustBePositive)
}

func testProposalSubmissionWithNegativeValidationTimestampFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: test.RandomNegativeI64(),
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"), commands.ErrMustBePositiveOrZero)
}

func testProposalSubmissionWithPositiveValidationTimestampSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: test.RandomPositiveI64(),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.validation_timestamp"), commands.ErrIsRequired)
}

func testProposalSubmissionWithClosingTimestampAfterEnactmentTimestampFails(t *testing.T) {
	closingTime := test.RandomPositiveI64()
	enactmentTime := test.RandomPositiveI64Before(closingTime)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   closingTime,
			EnactmentTimestamp: enactmentTime,
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.closing_timestamp"),
		errors.New("cannot be after enactment time"),
	)
}

func testProposalSubmissionWithClosingTimestampBeforeEnactmentTimestampSucceeds(t *testing.T) {
	enactmentTime := test.RandomPositiveI64()
	closingTime := test.RandomPositiveI64Before(enactmentTime)

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   closingTime,
			EnactmentTimestamp: enactmentTime,
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.closing_timestamp"),
		errors.New("cannot be after enactment time"),
	)
}

func testProposalSubmissionWithClosingTimestampAtEnactmentTimestampSucceeds(t *testing.T) {
	enactmentTime := test.RandomPositiveI64()

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:   enactmentTime,
			EnactmentTimestamp: enactmentTime,
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.closing_timestamp"),
		errors.New("cannot be after enactment time"),
	)
}

func testProposalSubmissionWithValidationTimestampAfterClosingTimestampFails(t *testing.T) {
	validationTime := test.RandomPositiveI64()
	closingTime := test.RandomPositiveI64Before(validationTime)
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    closingTime,
			ValidationTimestamp: validationTime,
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"),
		errors.New("cannot be after or equal to closing time"),
	)
}

func testProposalSubmissionWithValidationTimestampAtClosingTimestampFails(t *testing.T) {
	validationTime := test.RandomPositiveI64()

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    validationTime,
			ValidationTimestamp: validationTime,
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"),
		errors.New("cannot be after or equal to closing time"),
	)
}

func testProposalSubmissionWithValidationTimestampBeforeClosingTimestampSucceeds(t *testing.T) {
	closingTime := test.RandomPositiveI64()
	validationTime := test.RandomPositiveI64Before(closingTime)

	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    closingTime,
			ValidationTimestamp: validationTime,
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.validation_timestamp"),
		errors.New("cannot be after or equal to closing time"),
	)
}

func testProposalSubmissionWithoutRationalFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{})

	assert.Contains(t, err.Get("proposal_submission.rationale"), commands.ErrIsRequired)
}

func testProposalSubmissionWithRationalSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Rationale: &types.ProposalRationale{},
	})

	assert.Empty(t, err.Get("proposal_submission.rationale"))
}

func testProposalSubmissionWithRationalDescriptionSucceeds(t *testing.T) {
	tcs := []struct {
		name        string
		description string
	}{
		{
			name:        "with description of 10 characters",
			description: vgrand.RandomStr(10),
		}, {
			name:        "with description of 1024 characters",
			description: vgrand.RandomStr(1024),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Rationale: &types.ProposalRationale{
					Description: tc.description,
				},
			})

			assert.Empty(tt, err.Get("proposal_submission.rationale.description"))
		})
	}
}

func testProposalSubmissionWithIncorrectRationalDescriptionFails(t *testing.T) {
	tcs := []struct {
		name        string
		description string
		expectedErr error
	}{
		{
			name:        "with empty description",
			description: "",
			expectedErr: commands.ErrIsRequired,
		}, {
			name:        "with blank description",
			description: "     ",
			expectedErr: commands.ErrIsRequired,
		}, {
			name:        "with description > 1024",
			description: vgrand.RandomStr(20420),
			expectedErr: commands.ErrMustNotExceed20000Chars,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(&commandspb.ProposalSubmission{
				Rationale: &types.ProposalRationale{
					Description: tc.description,
				},
			})

			assert.Contains(tt, err.Get("proposal_submission.rationale.description"), tc.expectedErr)
		})
	}
}

func testProposalSubmissionWithRationalDescriptionAndTitleSucceeds(t *testing.T) {
	tcs := []struct {
		name       string
		shouldErr  bool
		submission *commandspb.ProposalSubmission
	}{
		{
			name: "NewMarket with rational Title and Description",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{},
				},
				Rationale: &types.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "NewMarket without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with UpdateMarket with rational Title and Description",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{},
				},
				Rationale: &types.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "with UpdateMarket without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with NewAsset with rational Title and Description",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{},
				},
				Rationale: &types.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "with NewAsset without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with UpdateNetworkParameter with rational Title and Description",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateNetworkParameter{},
				},
				Rationale: &types.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "with UpdateNetworkParameter without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateNetworkParameter{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with NewFreeform with rational Title and Description",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewFreeform{},
				},
				Rationale: &types.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(tc.submission)
			if !tc.shouldErr {
				assert.Empty(tt, err.Get("proposal_submission.rationale.title"), tc.name)
				assert.Empty(tt, err.Get("proposal_submission.rationale.description"), tc.name)
			} else {
				assert.Contains(tt, err.Get("proposal_submission.rationale.title"), commands.ErrIsRequired, tc.name)
				assert.Contains(tt, err.Get("proposal_submission.rationale.description"), commands.ErrIsRequired, tc.name)
			}
		})
	}
}

func checkProposalSubmission(cmd *commandspb.ProposalSubmission) commands.Errors {
	err := commands.CheckProposalSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
