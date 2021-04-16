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
