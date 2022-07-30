package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
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
	t.Run("Submitting a proposal with rational URL and hash succeeds", testProposalSubmissionWithRationalURLandHashSucceeds)
	t.Run("Submitting a proposal with missing rational URL or hash fails", testProposalSubmissionWithMissingRationalURLOrHashFails)
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
			value: RandomNegativeI64(),
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
			ClosingTimestamp: RandomPositiveI64(),
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
			value: RandomNegativeI64(),
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
			EnactmentTimestamp: RandomPositiveI64(),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.enactment_timestamp"), commands.ErrMustBePositive)
}

func testProposalSubmissionWithNegativeValidationTimestampFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: RandomNegativeI64(),
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.validation_timestamp"), commands.ErrMustBePositiveOrZero)
}

func testProposalSubmissionWithPositiveValidationTimestampSucceeds(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			ValidationTimestamp: RandomPositiveI64(),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.validation_timestamp"), commands.ErrIsRequired)
}

func testProposalSubmissionWithClosingTimestampAfterEnactmentTimestampFails(t *testing.T) {
	closingTime := RandomPositiveI64()
	enactmentTime := RandomPositiveI64Before(closingTime)
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
	enactmentTime := RandomPositiveI64()
	closingTime := RandomPositiveI64Before(enactmentTime)

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
	enactmentTime := RandomPositiveI64()

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
	validationTime := RandomPositiveI64()
	closingTime := RandomPositiveI64Before(validationTime)
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
	validationTime := RandomPositiveI64()

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
	closingTime := RandomPositiveI64()
	validationTime := RandomPositiveI64Before(closingTime)

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
			description: RandomStr(10),
		}, {
			name:        "with description of 1024 characters",
			description: RandomStr(1024),
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
			description: RandomStr(2042),
			expectedErr: commands.ErrMustNotExceed1024Chars,
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

func testProposalSubmissionWithRationalURLandHashSucceeds(t *testing.T) {
	tcs := []struct {
		name       string
		submission *commandspb.ProposalSubmission
	}{
		{
			name: "NewMarket with rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
					Url:  "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "NewMarket without rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with UpdateMarket with rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
					Url:  "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "with UpdateMarket without rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with NewAsset with rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
					Url:  "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "with NewAsset without rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with UpdateNetworkParameter with rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateNetworkParameter{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
					Url:  "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "with UpdateNetworkParameter without rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateNetworkParameter{},
				},
				Rationale: &types.ProposalRationale{},
			},
		}, {
			name: "with NewFreeform with rational URL and hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewFreeform{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
					Url:  "https://example.com/" + RandomStr(5),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(tc.submission)

			assert.Empty(tt, err.Get("proposal_submission.rationale.url"))
			assert.Empty(tt, err.Get("proposal_submission.rationale.hash"))
		})
	}
}

func testProposalSubmissionWithMissingRationalURLOrHashFails(t *testing.T) {
	tcs := []struct {
		name       string
		submission *commandspb.ProposalSubmission
	}{
		{
			name: "NewMarket with rational URL and no hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{},
				},
				Rationale: &types.ProposalRationale{
					Url: "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "NewMarket with rational hash and no URL",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewMarket{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
				},
			},
		}, {
			name: "with UpdateMarket with rational URL and no hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{},
				},
				Rationale: &types.ProposalRationale{
					Url: "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "with UpdateMarket with rational hash and no URL",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateMarket{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
				},
			},
		}, {
			name: "with NewAsset with rational with URL and no hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{},
				},
				Rationale: &types.ProposalRationale{
					Url: "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "with NewAsset without rational hash and no URL",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_NewAsset{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
				},
			},
		}, {
			name: "with UpdateNetworkParameter with rational URL and no hash",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateNetworkParameter{},
				},
				Rationale: &types.ProposalRationale{
					Url: "https://example.com/" + RandomStr(5),
				},
			},
		}, {
			name: "with UpdateNetworkParameter without rational hash and no URL",
			submission: &commandspb.ProposalSubmission{
				Terms: &types.ProposalTerms{
					Change: &types.ProposalTerms_UpdateNetworkParameter{},
				},
				Rationale: &types.ProposalRationale{
					Hash: RandomStr(10),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkProposalSubmission(tc.submission)

			if len(tc.submission.Rationale.Url) == 0 {
				assert.Contains(tt, err.Get("proposal_submission.rationale.url"), commands.ErrIsRequired)
				assert.Empty(tt, err.Get("proposal_submission.rationale.hash"))
			} else {
				assert.Contains(tt, err.Get("proposal_submission.rationale.hash"), commands.ErrIsRequired)
				assert.Empty(tt, err.Get("proposal_submission.rationale.url"))
			}
		})
	}
}

func checkProposalSubmission(cmd *commandspb.ProposalSubmission) commands.Errors {
	err := commands.CheckProposalSubmission(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
