// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/libs/test"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckBatchProposalSubmission(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilBatchProposalSubmissionFails)
	t.Run("Submitting a proposal without terms fails", testBatchProposalSubmissionWithoutTermsFails)
	t.Run("Submitting a proposal change without change fails", testBatchProposalSubmissionWithoutChangesFails)
	t.Run("Submitting a proposal without rational fails", testBatchProposalSubmissionWithoutRationalFails)
	t.Run("Submitting a proposal with rational succeeds", testBatchProposalSubmissionWithRationalSucceeds)
	t.Run("Submitting a proposal with rational description succeeds", testBatchProposalSubmissionWithRationalDescriptionSucceeds)
	t.Run("Submitting a proposal with incorrect rational description fails", testBatchProposalSubmissionWithIncorrectRationalDescriptionFails)
	t.Run("Submitting a proposal with rational URL and hash succeeds", testBatchProposalSubmissionWithRationalDescriptionAndTitleSucceeds)
	t.Run("Submitting a proposal with non-positive closing timestamp fails", testBatchProposalSubmissionWithNonPositiveClosingTimestampFails)
	t.Run("Submitting a proposal with positive closing timestamp succeeds", testBatchProposalSubmissionWithPositiveClosingTimestampSucceeds)
	t.Run("Submitting a proposal with non-positive enactment timestamp fails", testBatchProposalSubmissionWithNonPositiveEnactmentTimestampFails)
	t.Run("Submitting a proposal with positive enactment timestamp succeeds", testBatchProposalSubmissionWithPositiveEnactmentTimestampSucceeds)
	t.Run("Submitting a proposal with closing timestamp after enactment timestamp fails", testBatchProposalSubmissionWithClosingTimestampAfterEnactmentTimestampFails)
	t.Run("Submitting a proposal with closing timestamp before enactment timestamp succeeds", testBatchProposalSubmissionWithClosingTimestampBeforeEnactmentTimestampSucceeds)
	t.Run("Submitting a proposal with closing timestamp at enactment timestamp succeeds", testProposalSubmissionWithClosingTimestampAtEnactmentTimestampSucceeds)
}

func testNilBatchProposalSubmissionFails(t *testing.T) {
	err := checkBatchProposalSubmission(nil)

	assert.Contains(t, err.Get("batch_proposal_submission"), commands.ErrIsRequired)
}

func testBatchProposalSubmissionWithoutTermsFails(t *testing.T) {
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{})

	assert.Contains(t, err.Get("batch_proposal_submission.terms"), commands.ErrIsRequired)
}

func testBatchProposalSubmissionWithoutChangesFails(t *testing.T) {
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
		Terms: &commandspb.BatchProposalSubmissionTerms{},
	})

	assert.Contains(t, err.Get("batch_proposal_submission.terms.changes"), commands.ErrIsRequired)
}

func testBatchProposalSubmissionWithoutRationalFails(t *testing.T) {
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{})

	assert.Contains(t, err.Get("batch_proposal_submission.rationale"), commands.ErrIsRequired)
}

func testBatchProposalSubmissionWithRationalSucceeds(t *testing.T) {
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
		Rationale: &vegapb.ProposalRationale{},
	})

	assert.Empty(t, err.Get("batch_proposal_submission.rationale"))
}

func testBatchProposalSubmissionWithRationalDescriptionSucceeds(t *testing.T) {
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
			err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
				Rationale: &vegapb.ProposalRationale{
					Description: tc.description,
				},
			})

			assert.Empty(tt, err.Get("batch_proposal_submission.rationale.description"))
		})
	}
}

func testBatchProposalSubmissionWithIncorrectRationalDescriptionFails(t *testing.T) {
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
			err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
				Rationale: &vegapb.ProposalRationale{
					Description: tc.description,
				},
			})

			assert.Contains(tt, err.Get("batch_proposal_submission.rationale.description"), tc.expectedErr)
		})
	}
}

func testBatchProposalSubmissionWithRationalDescriptionAndTitleSucceeds(t *testing.T) {
	tcs := []struct {
		name       string
		shouldErr  bool
		submission *commandspb.BatchProposalSubmission
	}{
		{
			name: "NewMarket with rational Title and Description",
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_NewMarket{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "NewMarket without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_NewMarket{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{},
			},
		}, {
			name: "with UpdateMarket with rational Title and Description",
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_UpdateMarket{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "with UpdateMarket without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_UpdateMarket{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{},
			},
		}, {
			name: "with UpdateNetworkParameter with rational Title and Description",
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_UpdateNetworkParameter{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		}, {
			name:      "with UpdateNetworkParameter without rational Title and Description",
			shouldErr: true,
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_UpdateNetworkParameter{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{},
			},
		}, {
			name: "with NewFreeform with rational Title and Description",
			submission: &commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					Changes: []*vegapb.BatchProposalTermsChange{
						{
							Change: &vegapb.BatchProposalTermsChange_NewFreeform{},
						},
					},
				},
				Rationale: &vegapb.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			err := checkBatchProposalSubmission(tc.submission)
			if !tc.shouldErr {
				assert.Empty(tt, err.Get("batch_proposal_submission.rationale.title"), tc.name)
				assert.Empty(tt, err.Get("batch_proposal_submission.rationale.description"), tc.name)
			} else {
				assert.Contains(tt, err.Get("batch_proposal_submission.rationale.title"), commands.ErrIsRequired, tc.name)
				assert.Contains(tt, err.Get("batch_proposal_submission.rationale.description"), commands.ErrIsRequired, tc.name)
			}
		})
	}
}

func testBatchProposalSubmissionWithNonPositiveClosingTimestampFails(t *testing.T) {
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
			err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					ClosingTimestamp: tc.value,
					Changes:          []*vegapb.BatchProposalTermsChange{{}},
				},
				Rationale: &vegapb.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			})

			assert.Contains(t, err.Get("batch_proposal_submission.terms.closing_timestamp"), commands.ErrMustBePositive)
		})
	}
}

func testBatchProposalSubmissionWithPositiveClosingTimestampSucceeds(t *testing.T) {
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
		Terms: &commandspb.BatchProposalSubmissionTerms{
			ClosingTimestamp: test.RandomPositiveI64(),
			Changes:          []*vegapb.BatchProposalTermsChange{{}},
		},
		Rationale: &vegapb.ProposalRationale{
			Title:       vgrand.RandomStr(10),
			Description: vgrand.RandomStr(10),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.closing_timestamp"), commands.ErrMustBePositive)
}

func testBatchProposalSubmissionWithNonPositiveEnactmentTimestampFails(t *testing.T) {
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
			err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
				Terms: &commandspb.BatchProposalSubmissionTerms{
					ClosingTimestamp: test.RandomPositiveI64(),
					Changes: []*vegapb.BatchProposalTermsChange{{
						EnactmentTimestamp: tc.value,
					}},
				},
				Rationale: &vegapb.ProposalRationale{
					Title:       vgrand.RandomStr(10),
					Description: vgrand.RandomStr(10),
				},
			})

			assert.Contains(t, err.Get("batch_proposal_submission.terms.enactment_timestamp"), commands.ErrMustBePositive)
		})
	}
}

func testBatchProposalSubmissionWithPositiveEnactmentTimestampSucceeds(t *testing.T) {
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
		Terms: &commandspb.BatchProposalSubmissionTerms{
			ClosingTimestamp: test.RandomPositiveI64(),
			Changes: []*vegapb.BatchProposalTermsChange{{
				EnactmentTimestamp: test.RandomPositiveI64(),
			}},
		},
		Rationale: &vegapb.ProposalRationale{
			Title:       vgrand.RandomStr(10),
			Description: vgrand.RandomStr(10),
		},
	})

	assert.NotContains(t, err.Get("proposal_submission.terms.enactment_timestamp"), commands.ErrMustBePositive)
}

func testBatchProposalSubmissionWithClosingTimestampAfterEnactmentTimestampFails(t *testing.T) {
	closingTime := test.RandomPositiveI64()
	enactmentTime := test.RandomPositiveI64Before(closingTime)
	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
		Terms: &commandspb.BatchProposalSubmissionTerms{
			ClosingTimestamp: closingTime,
			Changes: []*vegapb.BatchProposalTermsChange{{
				EnactmentTimestamp: enactmentTime,
			}},
		},
		Rationale: &vegapb.ProposalRationale{
			Title:       vgrand.RandomStr(10),
			Description: vgrand.RandomStr(10),
		},
	})

	assert.Contains(t, err.Get("batch_proposal_submission.terms.closing_timestamp"),
		errors.New("cannot be after enactment time"),
	)
}

func testBatchProposalSubmissionWithClosingTimestampBeforeEnactmentTimestampSucceeds(t *testing.T) {
	enactmentTime := test.RandomPositiveI64()
	closingTime := test.RandomPositiveI64Before(enactmentTime)

	err := checkBatchProposalSubmission(&commandspb.BatchProposalSubmission{
		Terms: &commandspb.BatchProposalSubmissionTerms{
			ClosingTimestamp: closingTime,
			Changes: []*vegapb.BatchProposalTermsChange{{
				EnactmentTimestamp: enactmentTime,
			}},
		},
		Rationale: &vegapb.ProposalRationale{
			Title:       vgrand.RandomStr(10),
			Description: vgrand.RandomStr(10),
		},
	})

	assert.NotContains(t, err.Get("batch_proposal_submission.terms.closing_timestamp"),
		errors.New("cannot be after enactment time"),
	)
}

func checkBatchProposalSubmission(cmd *commandspb.BatchProposalSubmission) commands.Errors {
	err := commands.CheckBatchProposalSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
