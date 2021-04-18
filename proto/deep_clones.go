package proto

func (b BuiltinAsset) DeepClone() *BuiltinAsset {
	return &b
}

func (a AssetSource_BuiltinAsset) DeepClone() *AssetSource_BuiltinAsset {
	if a.BuiltinAsset != nil {
		a.BuiltinAsset = a.BuiltinAsset.DeepClone()
	}
	return &a
}

func (e ERC20) DeepClone() *ERC20 {
	return &e
}

func (a AssetSource_Erc20) DeepClone() *AssetSource_Erc20 {
	if a.Erc20 != nil {
		a.Erc20 = a.Erc20.DeepClone()
	}
	return &a
}

func (a AssetSource) DeepClone() *AssetSource {
	if a.Source != nil {
		switch src := a.Source.(type) {
		case *AssetSource_BuiltinAsset:
			a.Source = src.DeepClone()
		case *AssetSource_Erc20:
			a.Source = src.DeepClone()
		}
	}
	return &a
}

func (a Asset) DeepClone() *Asset {
	if a.Source != nil {
		switch src := a.Source.Source.(type) {
		case *AssetSource_BuiltinAsset:
			bia := *src.BuiltinAsset
			a.Source = &AssetSource{
				Source: &AssetSource_BuiltinAsset{
					BuiltinAsset: &bia,
				},
			}
		case *AssetSource_Erc20:
			erc := *src.Erc20
			a.Source = &AssetSource{
				Source: &AssetSource_Erc20{
					Erc20: &erc,
				},
			}
		}
	}
	return &a
}

func (n NetworkParameter) DeepClone() *NetworkParameter {
	return &n
}

func (u UpdateNetworkParameter) DeepClone() *UpdateNetworkParameter {
	if u.Changes != nil {
		u.Changes = u.Changes.DeepClone()
	}
	return &u
}

func (u UpdateMarket) DeepClone() *UpdateMarket {
	return &u
}

func (o OracleSpecToFutureBinding) DeepClone() *OracleSpecToFutureBinding {
	return &o
}

func (f FutureProduct) DeepClone() *FutureProduct {
	if f.OracleSpec != nil {
		f.OracleSpec = f.OracleSpec.DeepClone()
	}
	if f.OracleSpecBinding != nil {
		f.OracleSpecBinding = f.OracleSpecBinding.DeepClone()
	}
	return &f
}

func (i InstrumentConfiguration_Future) DeepClone() *InstrumentConfiguration_Future {
	if i.Future != nil {
		i.Future = i.Future.DeepClone()
	}
	return &i
}

func (i InstrumentConfiguration) DeepClone() *InstrumentConfiguration {
	if i.Product != nil {
		switch prod := i.Product.(type) {
		case *InstrumentConfiguration_Future:
			i.Product = prod.DeepClone()
		}
	}
	return &i
}

func (t TargetStakeParameters) DeepClone() *TargetStakeParameters {
	return &t
}

func (l LiquidityMonitoringParameters) DeepClone() *LiquidityMonitoringParameters {
	if l.TargetStakeParameters != nil {
		l.TargetStakeParameters = l.TargetStakeParameters.DeepClone()
	}
	return &l
}

func (p PriceMonitoringTrigger) DeepClone() *PriceMonitoringTrigger {
	return &p
}

func (p PriceMonitoringParameters) DeepClone() *PriceMonitoringParameters {
	if len(p.Triggers) > 0 {
		triggers := p.Triggers
		p.Triggers = make([]*PriceMonitoringTrigger, len(triggers))
		for i, t := range triggers {
			p.Triggers[i] = t.DeepClone()
		}
	}
	return &p
}

func (s SimpleModelParams) DeepClone() *SimpleModelParams {
	return &s
}

func (n NewMarketConfiguration_Simple) DeepClone() *NewMarketConfiguration_Simple {
	if n.Simple != nil {
		n.Simple = n.Simple.DeepClone()
	}
	return &n
}

func (l LogNormalModelParams) DeepClone() *LogNormalModelParams {
	return &l
}

func (l LogNormalRiskModel) DeepClone() *LogNormalRiskModel {
	if l.Params != nil {
		l.Params = l.Params.DeepClone()
	}
	return &l
}

func (n NewMarketConfiguration_LogNormal) DeepClone() *NewMarketConfiguration_LogNormal {
	if n.LogNormal != nil {
		n.LogNormal = n.LogNormal.DeepClone()
	}
	return &n
}

func (c ContinuousTrading) DeepClone() *ContinuousTrading {
	return &c
}

func (n NewMarketConfiguration_Continuous) DeepClone() *NewMarketConfiguration_Continuous {
	if n.Continuous != nil {
		n.Continuous = n.Continuous.DeepClone()
	}
	return &n
}

func (d DiscreteTrading) DeepClone() *DiscreteTrading {
	return &d
}

func (n NewMarketConfiguration_Discrete) DeepClone() *NewMarketConfiguration_Discrete {
	if n.Discrete != nil {
		n.Discrete = n.Discrete.DeepClone()
	}
	return &n
}

func (n NewMarketConfiguration) DeepClone() *NewMarketConfiguration {
	if n.Instrument != nil {
		n.Instrument = n.Instrument.DeepClone()
	}

	if n.LiquidityMonitoringParameters != nil {
		n.LiquidityMonitoringParameters = n.LiquidityMonitoringParameters.DeepClone()
	}

	if n.PriceMonitoringParameters != nil {
		n.PriceMonitoringParameters = n.PriceMonitoringParameters.DeepClone()
	}

	if n.RiskParameters != nil {
		switch risk := n.RiskParameters.(type) {
		case *NewMarketConfiguration_Simple:
			n.RiskParameters = risk.DeepClone()
		case *NewMarketConfiguration_LogNormal:
			n.RiskParameters = risk.DeepClone()
		}
	}

	if n.TradingMode != nil {
		switch trading := n.TradingMode.(type) {
		case *NewMarketConfiguration_Continuous:
			n.TradingMode = trading.DeepClone()
		case *NewMarketConfiguration_Discrete:
			n.TradingMode = trading.DeepClone()
		}
	}
	return &n
}

func (n NewMarketCommitment) DeepClone() *NewMarketCommitment {
	if len(n.Buys) > 0 {
		buys := n.Buys
		n.Buys = make([]*LiquidityOrder, len(buys))
		for i, lo := range buys {
			n.Buys[i] = lo.DeepClone()
		}
	}

	if len(n.Sells) > 0 {
		sells := n.Sells
		n.Sells = make([]*LiquidityOrder, len(sells))
		for i, lo := range sells {
			n.Sells[i] = lo.DeepClone()
		}
	}
	return &n
}

func (n NewMarket) DeepClone() *NewMarket {
	if n.Changes != nil {
		n.Changes = n.Changes.DeepClone()
	}
	if n.LiquidityCommitment != nil {
		n.LiquidityCommitment = n.LiquidityCommitment.DeepClone()
	}
	return &n
}

func (p ProposalTerms_UpdateNetworkParameter) DeepClone() *ProposalTerms_UpdateNetworkParameter {
	if p.UpdateNetworkParameter != nil {
		p.UpdateNetworkParameter = p.UpdateNetworkParameter.DeepClone()
	}
	return &p
}

func (p ProposalTerms_UpdateMarket) DeepClone() *ProposalTerms_UpdateMarket {
	if p.UpdateMarket != nil {
		p.UpdateMarket = p.UpdateMarket.DeepClone()
	}
	return &p
}

func (p ProposalTerms_NewMarket) DeepClone() *ProposalTerms_NewMarket {
	if p.NewMarket != nil {
		p.NewMarket = p.NewMarket.DeepClone()
	}
	return &p
}
func (n NewAsset) DeepClone() *NewAsset {
	if n.Changes != nil {
		n.Changes = n.Changes.DeepClone()
	}
	return &n
}

func (p ProposalTerms_NewAsset) DeepClone() *ProposalTerms_NewAsset {
	if p.NewAsset != nil {
		p.NewAsset = p.NewAsset.DeepClone()
	}
	return &p
}

func (p ProposalTerms) DeepClone() *ProposalTerms {
	if p.Change != nil {
		switch change := p.Change.(type) {
		case *ProposalTerms_NewAsset:
			p.Change = change.DeepClone()
		case *ProposalTerms_NewMarket:
			p.Change = change.DeepClone()
		case *ProposalTerms_UpdateMarket:
			p.Change = change.DeepClone()
		case *ProposalTerms_UpdateNetworkParameter:
			p.Change = change.DeepClone()
		}
	}
	return &p
}

func (p Proposal) DeepClone() *Proposal {
	if p.Terms != nil {
		p.Terms = p.Terms.DeepClone()
	}
	return &p
}

func (l LiquidityOrder) DeepClone() *LiquidityOrder {
	return &l
}

func (l LiquidityOrderReference) DeepClone() *LiquidityOrderReference {
	if l.LiquidityOrder != nil {
		l.LiquidityOrder = l.LiquidityOrder.DeepClone()
	}
	return &l
}

func (l LiquidityProvision) DeepClone() *LiquidityProvision {
	buys := l.Buys
	sells := l.Sells
	l.Buys = make([]*LiquidityOrderReference, len(l.Buys))
	l.Sells = make([]*LiquidityOrderReference, len(l.Sells))

	// Manually copy the pointer objects across
	for i, lor := range buys {
		l.Buys[i] = lor.DeepClone()
	}

	for i, lor := range sells {
		l.Sells[i] = lor.DeepClone()
	}
	return &l
}

func (p PriceMonitoringBounds) DeepClone() *PriceMonitoringBounds {
	if p.Trigger != nil {
		p.Trigger = p.Trigger.DeepClone()
	}
	return &p
}

func (l LiquidityProviderFeeShare) DeepClone() *LiquidityProviderFeeShare {
	return &l
}

func (m MarketData) DeepClone() *MarketData {
	if len(m.PriceMonitoringBounds) > 0 {
		pmb := m.PriceMonitoringBounds
		m.PriceMonitoringBounds = make([]*PriceMonitoringBounds, len(pmb))
		for i, p := range pmb {
			m.PriceMonitoringBounds[i] = p.DeepClone()
		}
	}

	if len(m.LiquidityProviderFeeShare) > 0 {
		lpfs := m.LiquidityProviderFeeShare
		m.LiquidityProviderFeeShare = make([]*LiquidityProviderFeeShare, len(lpfs))
		for i, l := range lpfs {
			m.LiquidityProviderFeeShare[i] = l.DeepClone()
		}
	}
	return &m
}

func (f Future) DeepClone() *Future {
	if f.OracleSpec != nil {
		f.OracleSpec = f.OracleSpec.DeepClone()
	}
	if f.OracleSpecBinding != nil {
		f.OracleSpecBinding = f.OracleSpecBinding.DeepClone()
	}
	return &f
}

func (i Instrument_Future) DeepClone() *Instrument_Future {
	if i.Future != nil {
		i.Future = i.Future.DeepClone()
	}
	return &i
}

func (i InstrumentMetadata) DeepClone() *InstrumentMetadata {
	if len(i.Tags) > 0 {
		tags := i.Tags
		i.Tags = make([]string, len(tags))
		for x, s := range tags {
			i.Tags[x] = s
		}
	}
	return &i
}

func (i Instrument) DeepClone() *Instrument {
	if i.Metadata != nil {
		i.Metadata = i.Metadata.DeepClone()
	}
	if i.Product != nil {
		switch prod := i.Product.(type) {
		case *Instrument_Future:
			i.Product = prod.DeepClone()
		}
	}
	return &i
}

func (s ScalingFactors) DeepClone() *ScalingFactors {
	return &s
}

func (m MarginCalculator) DeepClone() *MarginCalculator {
	if m.ScalingFactors != nil {
		m.ScalingFactors = m.ScalingFactors.DeepClone()
	}
	return &m
}

func (s SimpleRiskModel) DeepClone() *SimpleRiskModel {
	if s.Params != nil {
		s.Params = s.Params.DeepClone()
	}
	return &s
}

func (t TradableInstrument_SimpleRiskModel) DeepClone() *TradableInstrument_SimpleRiskModel {
	if t.SimpleRiskModel != nil {
		t.SimpleRiskModel = t.SimpleRiskModel.DeepClone()
	}
	return &t
}

func (t TradableInstrument_LogNormalRiskModel) DeepClone() *TradableInstrument_LogNormalRiskModel {
	if t.LogNormalRiskModel != nil {
		t.LogNormalRiskModel = t.LogNormalRiskModel.DeepClone()
	}
	return &t
}

func (t TradableInstrument) DeepClone() *TradableInstrument {
	if t.Instrument != nil {
		t.Instrument = t.Instrument.DeepClone()
	}

	if t.MarginCalculator != nil {
		t.MarginCalculator = t.MarginCalculator.DeepClone()
	}

	if t.RiskModel != nil {
		switch risk := t.RiskModel.(type) {
		case *TradableInstrument_SimpleRiskModel:
			t.RiskModel = risk.DeepClone()
		case *TradableInstrument_LogNormalRiskModel:
			t.RiskModel = risk.DeepClone()
		}
	}

	return &t
}

func (a AuctionDuration) DeepClone() *AuctionDuration {
	return &a
}

func (m Market_Continuous) DeepClone() *Market_Continuous {
	if m.Continuous != nil {
		m.Continuous = m.Continuous.DeepClone()
	}
	return &m
}

func (m Market_Discrete) DeepClone() *Market_Discrete {
	if m.Discrete != nil {
		m.Discrete = m.Discrete.DeepClone()
	}
	return &m
}

func (p PriceMonitoringSettings) DeepClone() *PriceMonitoringSettings {
	if p.Parameters != nil {
		p.Parameters = p.Parameters.DeepClone()
	}
	return &p
}

func (m MarketTimestamps) DeepClone() *MarketTimestamps {
	return &m
}

func (f FeeFactors) DeepClone() *FeeFactors {
	return &f
}

func (f Fees) DeepClone() *Fees {
	if f.Factors != nil {
		f.Factors = f.Factors.DeepClone()
	}
	return &f
}

func (m Market) DeepClone() *Market {
	if m.TradableInstrument != nil {
		m.TradableInstrument = m.TradableInstrument.DeepClone()
	}

	if m.Fees != nil {
		m.Fees = m.Fees.DeepClone()
	}

	if m.OpeningAuction != nil {
		m.OpeningAuction = m.OpeningAuction.DeepClone()
	}

	if m.TradingModeConfig != nil {
		switch tmc := m.TradingModeConfig.(type) {
		case *Market_Continuous:
			m.TradingModeConfig = tmc.DeepClone()
		case *Market_Discrete:
			m.TradingModeConfig = tmc.DeepClone()
		}
	}

	if m.PriceMonitoringSettings != nil {
		m.PriceMonitoringSettings = m.PriceMonitoringSettings.DeepClone()
	}

	if m.LiquidityMonitoringParameters != nil {
		m.LiquidityMonitoringParameters = m.LiquidityMonitoringParameters.DeepClone()
	}

	if m.MarketTimestamps != nil {
		m.MarketTimestamps = m.MarketTimestamps.DeepClone()
	}
	return &m
}

func (n NodeSignature) DeepClone() *NodeSignature {
	if len(n.Sig) > 0 {
		sigBytes := n.Sig
		n.Sig = make([]byte, len(sigBytes))
		for i, b := range sigBytes {
			n.Sig[i] = b
		}
	}
	return &n
}
