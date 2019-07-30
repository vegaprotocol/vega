# Matching engine

All docs regarding matching engine will live here...

## GetClosePNL

This call will calculate the _"cost"_ to the trader should the be closed out (based on current position). The _actual_ position should be passed, along with a `Side`. If the trader holds a long position, the call should be:

```go
closeOutPNL := matchingEngine.GetClosePNL(position.Size(), types.Side_Sell)
```

Internally, the matching engine will iterate over the orderbook (buy/sell side depending on the second argument). The _"cheapest"_ orders will be used first. This means that, for the buy side, the price levels are traversed backwards (the levels are tracked in descending order). The sell side is stored in ascending order, and is traversed as-is.

### TODO

It would be a good/easy performance win to memoise these values, given that we'll have to calculate these values for each block.

This call has no effect on the order status (ie it won't change the `Remaining` field of the orders).
