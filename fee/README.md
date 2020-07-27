Fee
===

This package covers fee handling in the vega protocol.

Fees are paid with every trade, for which we collect different fees:
- Maker fee, a fee being paid to the non-aggressive party in the trade
- Infrastructure fee, a fee being paid to maintain the vega network
- Liquidity fee, a fee being paid to the market makers.

Fees are calculate in the same way all the time, the market framework provide factors for each fees, these factore are applied to the trade ((trade.Price * trade.Size) * fee.Factor).

The engine provide multiple method, which will based on the trading mode, or the state of the traders taking part of the trade (e.g: distressed trader, etc), will split the calculated fee to be paid in between each parties, e.g:
in Continuous trading mode, all the fees are paid by the aggressive party.
