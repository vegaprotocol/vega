package vega

// IsEvent methods needs to be implemented so we can used mapped types in GQL union
func (Asset) IsEvent()                                                    {}
func (LiquidityProvision) IsEvent()                                       {}
func (Vote) IsEvent()                                                     {}
func (Order) IsEvent()                                                    {}
func (Account) IsEvent()                                                  {}
func (Trade) IsEvent()                                                    {}
func (Party) IsEvent()                                                    {}
func (MarginLevels) IsEvent()                                             {}
func (MarketData) IsEvent()                                               {}
func (GovernanceData) IsEvent()                                           {}
func (RiskFactor) IsEvent()                                               {}
func (Deposit) IsEvent()                                                  {}
func (Withdrawal) IsEvent()                                               {}
func (Market) IsEvent()                                                   {}
func (Future) IsProduct()                                                 {}
func (NewMarket) IsProposalChange()                                       {}
func (NewAsset) IsProposalChange()                                        {}
func (UpdateMarket) IsProposalChange()                                    {}
func (UpdateNetworkParameter) IsProposalChange()                          {}
func (NewFreeform) IsProposalChange()                                     {}
func (LogNormalRiskModel) IsRiskModel()                                   {}
func (SimpleRiskModel) IsRiskModel()                                      {}
func (SimpleModelParams) IsRiskModel()                                    {}
func (UpdateMarketConfiguration_Simple) IsUpdateMarketRiskParameters()    {}
func (UpdateMarketConfiguration_LogNormal) IsUpdateMarketRiskParameters() {}
