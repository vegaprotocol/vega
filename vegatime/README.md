# Time

This simple package handles a complex issue - how do we represent time in a network with geographically distributed nodes? The answer within the core is to rely on the block time that Tendermint provides.

The general rule of thumb for Vega developers is:
- Use Vega time for everything in Core
- Unless you are logging metrics, in which case use the *system* time

## Vega time
Vega time is set whenever the Tendermint calls Vega's [`BeginBlock`](https://github.com/vegaprotocol/vega/blob/fe5bf912ba1dc3b064b809048c3d192020819328/blockchain/tm/abci.go#L126-L128) function. As per [Tendermint's BeginBlock documentation](https://docs.tendermint.com/master/spec/abci/abci.html#beginblock), this includes the header field `Time`:

> **Time ([google.protobuf.Timestamp](https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/timestamp.proto))**: Time of the previous block. For heights > 1, it's the weighted median of the timestamps of the valid votes in the block.LastCommit. For height == 1, it's genesis time.

## System time
System time in this case means the current OS time on the computer. As noted above, the block time is set based on a median of the validators - meaning that your system time is likely to be somewhat different to the block time. This isn't a problem, as long API outputs and timestamps that go on chain use a consistent source of time. 

For logs that are more like system logs, or metrics that you may view as the operator of a node, it makes more sense for these to be in your timezone, to avoid having to work out what the block time was in your local time.
