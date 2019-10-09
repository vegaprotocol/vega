# Matching package 

A trade matching engine matches up order bids and offers to generate trades. Matching engines allocate trades among competing bids and offers at the same price.

## Calculating the cost of closing a trader out

This call will calculate the _"cost"_ to the trader should the be closed out (based on current position). The _actual_ position should be passed, along with a `Side`. If the trader holds a long position, the call should be:

```go
closeOutPNL := matchingEngine.GetClosePNL(position.Size(), types.Side_Sell)
```

Internally, the matching engine will iterate over the orderbook (buy/sell side depending on the second argument). The _"cheapest"_ orders will be used first. This means that, for the buy side, the price levels are traversed backwards (the levels are tracked in descending order). The sell side is stored in ascending order, and is traversed as-is.

This call has no effect on the order status (ie it won't change the `Remaining` field of the orders).
