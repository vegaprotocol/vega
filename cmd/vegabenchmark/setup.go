package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	ptypes "code.vegaprotocol.io/protos/vega"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/checkpoint"
	ethclient "code.vegaprotocol.io/vega/client/eth"
	"code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks"
	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/delegation"
	"code.vegaprotocol.io/vega/epochtime"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/genesis"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/netparams/checks"
	"code.vegaprotocol.io/vega/netparams/dispatch"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/processor"
	"code.vegaprotocol.io/vega/spam"
	"code.vegaprotocol.io/vega/staking"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/vegatime"
	"github.com/cenkalti/backoff"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	"github.com/prometheus/common/log"
)

func setupVega() (*processor.App, processor.Stats, error) {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())

	ctrl := gomock.NewController(&nopeTestReporter{log})
	notary := mocks.NewMockNotary(ctrl)
	oraclesAdaptors := mocks.NewMockOracleAdaptors(ctrl)

	ctx := context.Background()

	ethClient, err := ethclient.Dial(ctx, nodewallets.NewDefaultConfig().ETH.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise Ethereum client: %w", err)
	}

	commander := mocks.NewMockCommander(ctrl)
	commander.EXPECT().
		Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(nil)

	evtfwd := mocks.NewMockEvtForwarder(ctrl)
	evtfwd.EXPECT().Ack(gomock.Any()).AnyTimes().Return(true)

	oraclesM := mocks.NewMockOracleEngine(ctrl)
	oraclesM.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(oracles.SubscriptionID(1))

	governance := mocks.NewMockGovernanceEngine(ctrl)
	governance.EXPECT().OnChainTimeUpdate(gomock.Any(), gomock.Any()).AnyTimes()

	broker := mocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	timeService := vegatime.New(vegatime.NewDefaultConfig())

	vegaPaths, _ := vgtesting.NewVegaPaths()
	pass := vgrand.RandomStr(10)

	config := nodewallets.NewDefaultConfig()

	if _, err := nodewallets.GenerateEthereumWallet(config.ETH, vegaPaths, pass, pass, false); err != nil {
		return nil, nil, err
	}
	if _, err := nodewallets.GenerateVegaWallet(vegaPaths, pass, pass, false); err != nil {
		return nil, nil, err
	}
	nw, err := nodewallets.GetNodeWallets(config, vegaPaths, pass)
	if err != nil {
		return nil, nil, err
	}

	collateral := collateral.New(
		log,
		collateral.NewDefaultConfig(),
		broker,
		time.Time{},
	)
	assets := assets.New(
		log,
		assets.NewDefaultConfig(),
		nw,
		ethClient,
		timeService,
	)
	if err != nil {
		return nil, nil, err
	}
	topology := validators.NewTopology(
		log,
		validators.NewDefaultConfig(),
		nw.Vega,
		broker,
	)

	witness := validators.NewWitness(
		log,
		validators.NewDefaultConfig(),
		topology,
		commander,
		timeService,
	)

	banking := banking.New(
		log,
		banking.NewDefaultConfig(),
		collateral,
		witness,
		timeService,
		assets,
		notary,
		broker,
		topology,
	)

	exec := execution.NewEngine(
		log,
		execution.NewDefaultConfig(),
		timeService,
		collateral,
		oraclesM,
		broker,
	)

	genesisHandler := genesis.New(log, genesis.NewDefaultConfig())

	netp := netparams.New(
		log,
		netparams.NewDefaultConfig(),
		broker,
	)

	bstats := stats.NewBlockchain()

	epochService := epochtime.NewService(log, epochtime.NewDefaultConfig(), timeService, broker)

	netParams := netparams.New(log, netparams.NewDefaultConfig(), broker)

	stakingAccounts, _ := staking.New(
		log, staking.NewDefaultConfig(), broker, timeService, witness, ethClient, netParams,
	)

	delegationEngine := delegation.New(log, delegation.NewDefaultConfig(), broker, topology, stakingAccounts, epochService)
	netp.Watch(netparams.WatchParam{
		Param:   netparams.DelegationMinAmount,
		Watcher: delegationEngine.OnMinAmountChanged,
	})

	limits := mocks.NewMockLimits(ctrl)
	limits.EXPECT().CanTrade().AnyTimes().Return(true)
	limits.EXPECT().CanProposeMarket().AnyTimes().Return(true)
	limits.EXPECT().CanProposeAsset().AnyTimes().Return(true)

	spamEngine := spam.New(log, spam.NewDefaultConfig(), epochService, stakingAccounts)

	stakeV := mocks.NewMockStakeVerifier(ctrl)
	cp, _ := checkpoint.New(logging.NewTestLogger(), checkpoint.NewDefaultConfig())
	app := processor.NewApp(
		log,
		&paths.DefaultPaths{},
		processor.NewDefaultConfig(),
		func() {},
		assets,
		banking,
		broker,
		witness,
		evtfwd,
		exec,
		commander,
		genesisHandler,
		governance,
		notary,
		bstats,
		timeService,
		epochService,
		topology,
		netp,
		&processor.Oracle{
			Engine:   oraclesM,
			Adaptors: oraclesAdaptors,
		},
		delegationEngine,
		limits,
		stakeV,
		cp,
		spamEngine,
		nil,
	)
	err = registerExecutionCallbacks(log, netp, exec, assets, collateral)
	if err != nil {
		return nil, nil, err
	}

	// load markets and assets
	uponGenesisW := func(ctx context.Context, rawstate []byte) error {
		return uponGenesis(
			ctx,
			rawstate,
			log,
			assets,
			collateral,
			exec,
		)
	}

	setupGenesis(
		uponGenesisW,
		genesisHandler,
		timeService,
		netp,
		topology,
	)

	return app, bstats, nil
}

// UponGenesis loads all asset from genesis state.
func uponGenesis(
	ctx context.Context,
	rawstate []byte,
	log *logging.Logger,
	assetSvc *assets.Service,
	collateral *collateral.Engine,
	exec *execution.Engine,
) error {
	state, err := assets.LoadGenesisState(rawstate)
	if err != nil {
		return err
	}
	if state == nil {
		return nil
	}

	for k, v := range state {
		err := loadAsset(
			k, types.AssetDetailsFromProto(v),
			assetSvc, collateral,
		)
		if err != nil {
			return err
		}
	}

	mktscfg := []ptypes.Market{}
	for _, v := range markets {
		f, err := configsFS.ReadFile(v)
		if err != nil {
			return err
		}

		mkt := ptypes.Market{}
		err = jsonpb.Unmarshal(strings.NewReader(string(f)), &mkt)
		if err != nil {
			return fmt.Errorf("unable to unmarshal market configuration, %w", err)
		}
		mktscfg = append(mktscfg, mkt)
	}

	// then we load the markets
	for _, mkt := range mktscfg {
		mkt := mkt
		err = exec.SubmitMarket(ctx, types.MarketFromProto(&mkt))
		if err != nil {
			log.Panic("Unable to submit market", logging.Error(err))
		}
	}

	return nil
}

func loadAsset(
	id string,
	v *types.AssetDetails,
	assets *assets.Service,
	collateral *collateral.Engine,
) error {
	aid, err := assets.NewAsset(id, v)
	if err != nil {
		return fmt.Errorf("error instanciating asset %v", err)
	}

	asset, err := assets.Get(aid)
	if err != nil {
		return fmt.Errorf("unable to get asset %v", err)
	}

	// just a simple backoff here
	err = backoff.Retry(
		func() error {
			err := asset.Validate()
			if !asset.IsValid() {
				return err
			}
			return nil
		},
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5),
	)
	if err != nil {
		return fmt.Errorf("unable to instantiate new asset err=%v, asset-source=%s", err, v.String())
	}
	if err := assets.Enable(aid); err != nil {
		return fmt.Errorf("unable to enable asset: %v", err)
	}

	assetD := asset.Type()
	if err := collateral.EnableAsset(context.Background(), *assetD); err != nil {
		return fmt.Errorf("unable to enable asset in collateral: %v", err)
	}

	log.Info("new asset added successfully",
		logging.String("asset", asset.String()))

	return nil
}

func setupGenesis(
	uponGenesis func(ctx context.Context, rawstate []byte) error,
	genesisHandler *genesis.Handler,
	timeService *vegatime.Svc,
	netps *netparams.Store,
	topology *validators.Topology,
) {
	genesisHandler.OnGenesisTimeLoaded(timeService.SetTimeNow)
	genesisHandler.OnGenesisAppStateLoaded(
		uponGenesis,
		netps.UponGenesis,
		topology.LoadValidatorsOnGenesis,
	)
}

func registerExecutionCallbacks(
	log *logging.Logger,
	netps *netparams.Store,
	exec *execution.Engine,
	assets *assets.Service,
	collateral *collateral.Engine,
) error {
	if err := netps.AddRules(
		netparams.ParamStringRules(
			netparams.RewardAsset,
			checks.RewardAssetUpdate(log, assets, collateral),
		),
	); err != nil {
		return err
	}

	// now add some watcher for our netparams
	return netps.Watch(
		netparams.WatchParam{
			Param:   netparams.RewardAsset,
			Watcher: dispatch.RewardAssetUpdate(log, assets),
		},
		netparams.WatchParam{
			Param:   netparams.MarketMarginScalingFactors,
			Watcher: exec.OnMarketMarginScalingFactorsUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsMakerFee,
			Watcher: exec.OnMarketFeeFactorsMakerFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketFeeFactorsInfrastructureFee,
			Watcher: exec.OnMarketFeeFactorsInfrastructureFeeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityStakeToCCYSiskas,
			Watcher: exec.OnSuppliedStakeToObligationFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketValueWindowLength,
			Watcher: exec.OnMarketValueWindowLengthUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeScalingFactor,
			Watcher: exec.OnMarketTargetStakeScalingFactorUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketTargetStakeTimeWindow,
			Watcher: exec.OnMarketTargetStakeTimeWindowUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvidersFeeDistribitionTimeStep,
			Watcher: exec.OnMarketLiquidityProvidersFeeDistributionTimeStep,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityProvisionShapesMaxSize,
			Watcher: exec.OnMarketLiquidityProvisionShapesMaxSizeUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityMaximumLiquidityFeeFactorLevel,
			Watcher: exec.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityBondPenaltyParameter,
			Watcher: exec.OnMarketLiquidityBondPenaltyUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketLiquidityTargetStakeTriggeringRatio,
			Watcher: exec.OnMarketLiquidityTargetStakeTriggeringRatio,
		},
		netparams.WatchParam{
			Param:   netparams.MarketAuctionMinimumDuration,
			Watcher: exec.OnMarketAuctionMinimumDurationUpdate,
		},
		netparams.WatchParam{
			Param:   netparams.MarketProbabilityOfTradingTauScaling,
			Watcher: exec.OnMarketProbabilityOfTradingTauScalingUpdate,
		},
	)
}

type nopeTestReporter struct{ log *logging.Logger }

func (n *nopeTestReporter) Errorf(format string, args ...interface{}) {
	n.log.Errorf(format, args...)
}

func (n *nopeTestReporter) Fatalf(format string, args ...interface{}) {
	n.log.Errorf(format, args...)
}
