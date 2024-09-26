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
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/stretchr/testify/assert"
)

func TestSubmissionOfNewPAPWithoutDetailsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase"), commands.ErrIsRequired)
}

func TestSubmissionOfNewPAPWithoutChangeDetailsFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{},
			},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes"), commands.ErrIsRequired)
}

func TestSubmissionOfNewPAPWithoutFromAssetFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.from"), commands.ErrIsRequired)
}

func TestSubmissionOfNewPAPInvalidFromAccountTypeFails(t *testing.T) {
	validFromAccountTypes := map[types.AccountType]struct{}{
		types.AccountType_ACCOUNT_TYPE_BUY_BACK_FEES: {},
	}
	for tp := range types.AccountType_name {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
					NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
						Changes: &types.NewProtocolAutomatedPurchaseChanges{
							FromAccountType: types.AccountType(tp),
						},
					},
				},
			},
		})
		_, ok := validFromAccountTypes[types.AccountType(tp)]
		if !ok {
			if tp == 0 {
				assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.from_account_type"), commands.ErrIsRequired)
			} else {
				assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.from_account_type"), commands.ErrIsNotValid)
			}
		} else {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.from_account_type"), commands.ErrIsNotValid)
		}
	}
}

func TestSubmissionOfNewPAPInvalidToAccountTypeFails(t *testing.T) {
	validAccountTypes := map[types.AccountType]struct{}{
		types.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE: {},
		types.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD:    {},
		types.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY: {},
		types.AccountType_ACCOUNT_TYPE_BUY_BACK_FEES:    {},
	}
	for tp := range types.AccountType_name {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
					NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
						Changes: &types.NewProtocolAutomatedPurchaseChanges{
							ToAccountType: types.AccountType(tp),
						},
					},
				},
			},
		})
		_, ok := validAccountTypes[types.AccountType(tp)]
		if !ok {
			if tp == 0 {
				assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.to_account_type"), commands.ErrIsRequired)
			} else {
				assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.to_account_type"), commands.ErrIsNotValid)
			}
		} else {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.to_account_type"), commands.ErrIsNotValid)
		}
	}
}

func TestSubmissionOfNewPAPWithoutMarketIDFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.market_id"), commands.ErrIsRequired)
}

func TestSubmissionOfNewPAPWithInvalidPriceOffsetFails(t *testing.T) {
	values := []string{"", "banana", "-1", "0", "0.9", "1.1"}
	errors := []error{commands.ErrIsRequired, commands.ErrNotAValidFloat, commands.ErrMustBePositive, commands.ErrMustBePositive, nil, nil}
	for i, v := range values {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
					NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
						Changes: &types.NewProtocolAutomatedPurchaseChanges{
							OracleOffsetFactor: v,
						},
					},
				},
			},
		})
		if errors[i] != nil {
			assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.oracle_offset_factor"), errors[i])
		} else {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.oracle_offset_factor"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.oracle_offset_factor"), commands.ErrNotAValidFloat)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.oracle_offset_factor"), commands.ErrMustBePositive)
		}
	}
}

func TestSubmissionOfNewPAPWithInvalidDurationFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionDuration: "",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_duration"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionDuration: "dsfhjkhkj",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_duration"), fmt.Errorf("must be a valid duration"))

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionDuration: "10h5m",
					},
				},
			},
		},
	})
	assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_duration"), fmt.Errorf("must be a valid duration"))
	assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_duration"), commands.ErrIsRequired)
}

func TestSubmissionOfNewPAPWithInvalidMinMaxFails(t *testing.T) {
	values := []string{"", "banana", "-1", "0", "1000"}
	errors := []error{commands.ErrIsRequired, commands.ErrMustBePositive, commands.ErrMustBePositive, commands.ErrMustBePositive, nil}

	for i, v := range values {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
					NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
						Changes: &types.NewProtocolAutomatedPurchaseChanges{
							MinimumAuctionSize: v,
							MaximumAuctionSize: v,
						},
					},
				},
			},
		})
		if errors[i] != nil {
			assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.minimum_auction_size"), errors[i])
			assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.maximum_auction_size"), errors[i])
		} else {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.minimum_auction_size"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.minimum_auction_size"), commands.ErrMustBePositive)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.maximum_auction_size"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.maximum_auction_size"), commands.ErrMustBePositive)
		}
	}
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						MinimumAuctionSize: "100",
						MaximumAuctionSize: "99",
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.maximum_auction_size"), fmt.Errorf("must be greater than or equal to minimum_auction_size"))
}

func TestSubmissionOfNewPAPWithInvalidExpiry(t *testing.T) {
	values := []int64{-1, 0, 100}
	errors := []error{commands.ErrMustBePositiveOrZero, nil, nil}

	for i, v := range values {
		err := checkProposalSubmission(&commandspb.ProposalSubmission{
			Terms: &types.ProposalTerms{
				Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
					NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
						Changes: &types.NewProtocolAutomatedPurchaseChanges{
							ExpiryTimestamp: v,
						},
					},
				},
			},
		})
		if errors[i] != nil {
			assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.expiry_timestamp"), errors[i])
		} else {
			assert.NotContains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.expiry_timestamp"), errors[i])
		}
	}
}

func TestSubmissionInvalidAuctionScheduleFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: nil,
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_schedule"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_External{},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_schedule"), fmt.Errorf("auction schedule must be an internal time trigger"))

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_Time{},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_schedule"), fmt.Errorf("auction schedule must be an internal time trigger"))

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_schedule.internal.timetrigger"), commands.ErrOneTimeTriggerAllowedMax)

	initial1 := int64(100)
	initial2 := int64(200)
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{
												{
													Initial: &initial1,
												},
												{
													Initial: &initial2,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_schedule.internal.timetrigger"), commands.ErrOneTimeTriggerAllowedMax)

	initial1 = -100
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{
												{
													Initial: &initial1,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_schedule.internal.timetrigger.triggers.0.initial"), commands.ErrIsNotValid)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{
												{
													Every: initial1,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_schedule.internal.timetrigger.triggers.0.every"), commands.ErrIsNotValid)
}

func TestSubmissionInvalidVolumeSnapshotScheduleFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: nil,
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_volume_snapshot_schedule"), commands.ErrIsRequired)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_External{},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_volume_snapshot_schedule"), fmt.Errorf("auction volume snapshot schedule must be an internal time trigger"))

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_Time{},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.auction_volume_snapshot_schedule"), fmt.Errorf("auction volume snapshot schedule must be an internal time trigger"))

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_volume_snapshot_schedule.internal.timetrigger"), commands.ErrOneTimeTriggerAllowedMax)

	initial1 := int64(100)
	initial2 := int64(200)
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{
												{
													Initial: &initial1,
												},
												{
													Initial: &initial2,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_volume_snapshot_schedule.internal.timetrigger"), commands.ErrOneTimeTriggerAllowedMax)

	initial1 = -100
	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{
												{
													Initial: &initial1,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_volume_snapshot_schedule.internal.timetrigger.triggers.0.initial"), commands.ErrIsNotValid)

	err = checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewProtocolAutomatedPurchase{
				NewProtocolAutomatedPurchase: &types.NewProtocolAutomatedPurchase{
					Changes: &types.NewProtocolAutomatedPurchaseChanges{
						AuctionVolumeSnapshotSchedule: &types.DataSourceDefinition{
							SourceType: &types.DataSourceDefinition_Internal{
								Internal: &types.DataSourceDefinitionInternal{
									SourceType: &types.DataSourceDefinitionInternal_TimeTrigger{
										TimeTrigger: &types.DataSourceSpecConfigurationTimeTrigger{
											Triggers: []*datapb.InternalTimeTrigger{
												{
													Every: initial1,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	assert.Contains(t, err.Get("proposal_submission.terms.change.protocol_automated_purchase.changes.data_spec_for_auction_volume_snapshot_schedule.internal.timetrigger.triggers.0.every"), commands.ErrIsNotValid)
}
