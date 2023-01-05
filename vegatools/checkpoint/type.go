package checkpoint

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/protos/vega"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	events "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/sha3"
)

type all struct {
	messages map[string]proto.Message
}

func newAll() *all {
	return &all{
		messages: map[string]proto.Message{
			"governance":         new(checkpoint.Proposals),
			"assets":             new(checkpoint.Assets),
			"collateral":         new(checkpoint.Collateral),
			"network_parameters": new(checkpoint.NetParams),
			"delegation":         new(checkpoint.Delegate),
			"epoch":              new(events.EpochEvent),
			"block":              new(checkpoint.Block),
			"rewards":            new(checkpoint.Rewards),
			"banking":            new(checkpoint.Banking),
			"validators":         new(checkpoint.Validators),
			"staking":            new(checkpoint.Staking),
			"multisig_control":   new(checkpoint.MultisigControl),
			"market_tracker":     new(checkpoint.MarketTracker),
		},
	}
}

type allJSON map[string]json.RawMessage

// AssetErr a convenience error type.
type AssetErr []error

func (a *all) CheckAssetsCollateral() error {
	assets, ok := a.messages["assets"].(*checkpoint.Assets)
	if !ok {
		return fmt.Errorf("assets not found")
	}

	assetIDSet := make(map[string]struct{}, len(assets.Assets))
	for _, ass := range assets.Assets {
		assetIDSet[ass.Id] = struct{}{}
	}

	cAssets := make(map[string]struct{}, len(assetIDSet)) // should be no more than total assets
	for _, c := range a.messages["collateral"].(*checkpoint.Collateral).Balances {
		cAssets[c.Asset] = struct{}{}
	}

	var errs []error

	for ca := range cAssets {
		if _, ok := assetIDSet[ca]; !ok {
			errs = append(errs, fmt.Errorf("collateral contains '%s' asset, asset checkpoint does not", ca))
		}
	}

	if len(errs) != 0 {
		return AssetErr(errs)
	}

	return nil
}

func (a *all) JSON() ([]byte, error) {
	// format nicely
	marshaler := jsonpb.Marshaler{
		Indent: "   ",
	}

	allJsn := allJSON{}

	for k, v := range a.messages {
		var buf bytes.Buffer
		if err := marshaler.Marshal(&buf, v); err != nil {
			return nil, err
		}
		allJsn[k] = buf.Bytes()
	}

	b, err := json.MarshalIndent(allJsn, "", "   ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

// fromJSON can be used in the future to load JSON input and generate a checkpoint file.
func fromJSON(in []byte) (*all, error) {
	allJsn := allJSON{}
	if err := json.Unmarshal(in, &allJsn); err != nil {
		return nil, err
	}

	a := newAll()

	for k, v := range allJsn {
		reader := bytes.NewReader(v)
		if err := jsonpb.Unmarshal(reader, a.messages[k]); err != nil {
			return nil, err
		}
	}

	return a, nil
}

// hash returns the hash for a checkpoint (from core repo - needs to be kept in sync).
func hash(data []byte) string {
	h := sha3.New256()
	_, _ = h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func allBytes(cp *checkpoint.Checkpoint) []byte {
	buf := types.NewCheckpointFromProto(cp).HashBytes()
	return buf.Bytes()
}

func (a *all) CheckpointData() ([]byte, string, error) {
	if len(a.messages) == 0 {
		return nil, "", fmt.Errorf("no checkpoint data found")
	}

	cp := &checkpoint.Checkpoint{
		Governance:        []byte{0},
		Assets:            []byte{0},
		Collateral:        []byte{0},
		NetworkParameters: []byte{0},
		Delegation:        []byte{0},
		Epoch:             []byte{0},
		Block:             []byte{0},
		Rewards:           []byte{0},
		Banking:           []byte{0},
		Validators:        []byte{0},
		Staking:           []byte{0},
		MultisigControl:   []byte{0},
		MarketTracker:     []byte{0},
	}
	cp.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		name := string(fd.Name())
		v, ok := a.messages[name]
		if !ok {
			return true
		}
		msg, err := proto.Marshal(v)
		if err != nil {
			return true
		}
		cp.ProtoReflect().Set(fd, protoreflect.ValueOf(msg))
		return true
	})

	ret, err := proto.Marshal(cp)
	if err != nil {
		return nil, "", err
	}

	if len(ret) == 0 {
		return nil, "", fmt.Errorf("failed to parse checkpoint data")
	}

	return ret, hash(allBytes(cp)), nil
}

// Error outputs the mismatches in an easy to read way.
func (a AssetErr) Error() string {
	out := make([]string, 0, len(a)+1)
	out = append(out, "unexpected asset/collateral data found:")
	for _, e := range a {
		out = append(out, fmt.Sprintf("\t%s", e.Error()))
	}
	return strings.Join(out, "\n")
}

func dummy() *all {
	ae := &checkpoint.AssetEntry{
		Id: "ETH",
		AssetDetails: &vega.AssetDetails{
			Name:     "ETH",
			Symbol:   "ETH",
			Decimals: 5,
			Quantum:  "",
			Source: &vega.AssetDetails_BuiltinAsset{
				BuiltinAsset: &vega.BuiltinAsset{
					MaxFaucetAmountMint: "100000000000",
				},
			},
		},
	}
	bal := &checkpoint.AssetBalance{
		Party:   "deadbeef007",
		Asset:   "ETH",
		Balance: "1000000",
	}
	prop := &vega.Proposal{
		Id:        "prop-1",
		Reference: "dummy-proposal",
		PartyId:   "deadbeef007",
		State:     vega.Proposal_STATE_ENACTED,
		Timestamp: time.Now().Add(-1 * time.Hour).Unix(),
		Terms: &vega.ProposalTerms{
			ClosingTimestamp:    time.Now().Add(24 * time.Hour).Unix(),
			EnactmentTimestamp:  time.Now().Add(-10 * time.Minute).Unix(),
			ValidationTimestamp: time.Now().Add(-1*time.Hour - time.Second).Unix(),
			Change: &vega.ProposalTerms_NewMarket{
				NewMarket: &vega.NewMarket{
					Changes: &vega.NewMarketConfiguration{
						Instrument: &vega.InstrumentConfiguration{
							Name: "ETH/FOO",
							Code: "bar",
							Product: &vega.InstrumentConfiguration_Future{
								Future: &vega.FutureProduct{ // omitted oracle spec for now
									SettlementAsset: "ETH",
									QuoteName:       "ETH",
								},
							},
						},
						DecimalPlaces: 5,
						PriceMonitoringParameters: &vega.PriceMonitoringParameters{
							Triggers: []*vega.PriceMonitoringTrigger{
								{
									Horizon:          10,
									Probability:      "0.95",
									AuctionExtension: 10,
								},
							},
						},
						LiquidityMonitoringParameters: &vega.LiquidityMonitoringParameters{
							TargetStakeParameters: &vega.TargetStakeParameters{
								TimeWindow:    10,
								ScalingFactor: 0.7,
							},
							TriggeringRatio:  "0.5",
							AuctionExtension: 10,
						},
						RiskParameters: &vega.NewMarketConfiguration_LogNormal{
							LogNormal: &vega.LogNormalRiskModel{
								RiskAversionParameter: 0.1,
								Tau:                   0.2,
								Params: &vega.LogNormalModelParams{
									Mu:    0.3,
									R:     0.3,
									Sigma: 0.3,
								},
							},
						},
						LpPriceRange: "0.95",
					},
				},
			},
		},
	}
	del := &checkpoint.Delegate{
		Active: []*checkpoint.DelegateEntry{
			{
				Party:    "deadbeef007",
				Node:     "node0",
				Amount:   "100",
				EpochSeq: 0,
			},
		},
		Pending: []*checkpoint.DelegateEntry{
			{
				Party:      "deadbeef007",
				Node:       "node0",
				Amount:     "100",
				Undelegate: true,
				EpochSeq:   1,
			},
		},
		AutoDelegation: []string{
			"deadbeef007",
		},
	}
	t := time.Now()
	return &all{
		messages: map[string]proto.Message{
			"assets": &checkpoint.Assets{
				Assets: []*checkpoint.AssetEntry{ae},
			},
			"collateral": &checkpoint.Collateral{
				Balances: []*checkpoint.AssetBalance{bal},
			},
			"governance": &checkpoint.Proposals{
				Proposals: []*vega.Proposal{prop},
			},
			"network_parameters": &checkpoint.NetParams{
				Params: []*vega.NetworkParameter{
					{
						Key:   "foo",
						Value: "bar",
					},
				},
			},
			"delegation": del,
			"epoch": &events.EpochEvent{
				Seq:        0,
				Action:     vega.EpochAction_EPOCH_ACTION_START,
				StartTime:  t.UnixNano(),
				ExpireTime: t.Add(24 * time.Hour).UnixNano(),
				EndTime:    t.Add(25 * time.Hour).UnixNano(),
			},
			"block": &checkpoint.Block{
				Height: 1,
			},
			"banking": &checkpoint.Banking{
				RecurringTransfers: &checkpoint.RecurringTransfers{
					RecurringTransfers: []*events.Transfer{
						{
							Id:              "someid",
							From:            "somefrom",
							FromAccountType: vega.AccountType_ACCOUNT_TYPE_GENERAL,
							To:              "someto",
							ToAccountType:   vega.AccountType_ACCOUNT_TYPE_GENERAL,
							Asset:           "someasset",
							Amount:          "100",
							Reference:       "someref",
							Status:          events.Transfer_STATUS_PENDING,
							Kind: &events.Transfer_Recurring{
								Recurring: &events.RecurringTransfer{
									StartEpoch: 10,
									EndEpoch:   func() *uint64 { e := uint64(100); return &e }(),
									Factor:     "1",
								},
							},
						},
					},
				},
			},
		},
	}
}
