package proto

func (lp *LiquidityProvision) DeepClone() *LiquidityProvision {
	// Shallow copy the native types
	cpy := &LiquidityProvision{}
	*cpy = *lp

	cpy.Buys = make([]*LiquidityOrderReference, len(lp.Buys))
	cpy.Sells = make([]*LiquidityOrderReference, len(lp.Sells))

	// Manually copy the pointer objects across
	for i, lor := range lp.Buys {
		tempBuy := *lor
		tempLO := *lor.LiquidityOrder
		tempBuy.LiquidityOrder = &tempLO
		cpy.Buys[i] = &tempBuy
	}

	for i, lor := range lp.Sells {
		tempSell := *lor
		tempLO := *lor.LiquidityOrder
		tempSell.LiquidityOrder = &tempLO
		cpy.Sells[i] = &tempSell
	}
	return cpy
}
