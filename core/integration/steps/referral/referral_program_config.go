package referral

type Config struct {
	BenefitTiersConfigs *benefitTiersConfig
	StakingTiersConfigs *stakingTiersConfig
}

func NewReferralProgramConfig() *Config {
	return &Config{
		BenefitTiersConfigs: newBenefitTiersConfigs(),
		StakingTiersConfigs: newStakingTiersConfigs(),
	}
}
