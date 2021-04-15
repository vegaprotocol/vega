package proto

func (a *Asset) DeepClone() *Asset {
	cpy := &Asset{}
	*cpy = *a

	if a.Source != nil {
		switch src := cpy.Source.Source.(type) {
		case *AssetSource_BuiltinAsset:
			bia := *src.BuiltinAsset
			cpy.Source = &AssetSource{
				Source: &AssetSource_BuiltinAsset{
					BuiltinAsset: &bia,
				},
			}
		case *AssetSource_Erc20:
			erc := *src.Erc20
			cpy.Source = &AssetSource{
				Source: &AssetSource_Erc20{
					Erc20: &erc,
				},
			}
		}
	}
	return cpy
}

func (l *LiquidityProvision) DeepClone() *LiquidityProvision {
	// Shallow copy the native types
	cpy := &LiquidityProvision{}
	*cpy = *l

	cpy.Buys = make([]*LiquidityOrderReference, len(l.Buys))
	cpy.Sells = make([]*LiquidityOrderReference, len(l.Sells))

	// Manually copy the pointer objects across
	for i, lor := range l.Buys {
		tempBuy := *lor
		tempLO := *lor.LiquidityOrder
		tempBuy.LiquidityOrder = &tempLO
		cpy.Buys[i] = &tempBuy
	}

	for i, lor := range l.Sells {
		tempSell := *lor
		tempLO := *lor.LiquidityOrder
		tempSell.LiquidityOrder = &tempLO
		cpy.Sells[i] = &tempSell
	}
	return cpy
}
