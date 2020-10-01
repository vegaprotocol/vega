package netparams

func defaultNetParams() map[string]Value {
	return map[string]Value{
		MarketMarginScalingFactorSearchLevel:       NewFloat(FloatGTE(0)).MustUpdate("1.1"),
		MarketMarginScalingFactorInitialMargin:     NewFloat(FloatGTE(0)).MustUpdate("1.2"),
		MarketMarginScalingFactorCollateralRelease: NewFloat(FloatGTE(0)).MustUpdate("1.4"),
		MarketFeeFactorsMakerFee:                   NewFloat(FloatGTE(0), FloatLTE(1)).MustUpdate("0.00025"),
		MarketFeeFactorsInfrastructureFee:          NewFloat(FloatGTE(0), FloatLTE(1)).MustUpdate("0.0005"),
		MarketFeeFactorsLiquidityFee:               NewFloat(FloatGTE(0), FloatLTE(1)).MustUpdate("0.001"),
	}
}
