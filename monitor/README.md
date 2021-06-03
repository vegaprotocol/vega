# Monitor package

This package contains all engines that will monitor the sanity of the markets currently running. If something goes awry, these engines will suspend the _"normal"_ trading mode of markets, and trigger an auction. What it means to change the way a market trades is beyond the domain of the engines, just like the market itself shouldn't be aware of how the determination is made whether or not an auction ought to be triggered.

That's where the `AuctionState` in this package comes in.

## Glorified DTO

At its core, the `AuctionState` is little more than a DTO. It keeps track of what trading mode a given market is in, what its default trading mode is, and why it is in the mode it is in. If a market opens, a new `AuctionState` object is created for it. The state will immediately be set to reflect a market in opening auction. The start and end times will be set, and the state object will have a flag set that this auction period has just been triggered.

The market will then check if its auction-state has been updated (`AuctionState.AuctionStart()`), and if it has, the market calls `EnterAuction()`. This function will do whatever a market has to do to enter an auction (update the orderbook, for example), and finally acknowledge the auction was started by calling `AuctionState.AuctionStarted()`. This function returns an auction event that the market will send to the broker.

Every `market.OnChainTimeUpdate()`, the market checks whether it is currently trading in an auction. If it's an opening auction, we're dealing with a simple time-limited auction, and the market will check to see whether or not the auction period has expired. If it has, the market calls `AuctionState.SetReadyToLeave()`, performs everything it has to do to terminate the auction (updating the orderbook, uncrossing auction orders, ...), and then finally `AuctionState.Left()` is called. Like the `AuctionStarted()` call, this returns an event to push to the broker.

## For the monitor sub-packages

Each monitor sub-package will have a slightly different interface definition for the `AuctionState` type. Specific calls like `StartPriceAuction()` are used by the price monitoring engine in case the mark price exceeds the expected range, but this call is meaningless to any other monitoring engines. What monitoring engines _can_ do, however is extend an auction period if needed. This is the reason why auction starts and endings require 2 calls (one returning a boolean value indicating an auction is starting/ending, the other for the market to confirm it has done so).

One monitoring engine could find that it's safe to end an auction it triggered, but we don't want to leave this auction if another monitoring engine would immediately trigger a new auction period. In code, it could look something like this:

```go
func (m *Market) inMarket() {
    if m.as.InAuction() {
        // see if price auction has anything to say/do with this auction
        m.price.CheckAuction(m.as)
        // same, but liquidity monitoring
        m.liquidity.CheckAuction(m.as)
        // any other monitoring engines we decide to add
        m.parties.CheckAuction(m.as)
        // monitoring engines have determined we can/should leave the auction
        if m.as.CanLeave() {
            m.LeaveAuction()
        }
    }
}
// in monitoring engines:
func (p *Price) CheckAuction(as AuctionState) {
    // clearly, this monitoring engine only cares about auctions it started
    if !as.IsPriceAuction() {
        return
    }
    if as.AuctionStart().Before(as.CanLeave()) {
        // auction has expired - mark auction as over
        // this can be determined in different ways, that's up to the engine itself
        // for example: auctions bound by traded volume
        as.SetReadyToLeave()
    }
    // auction carries on as per usual
}

func (l *Liquidity) CheckAuction(as AuctionSate) {
    if as.IsOpeningAuction() {
        // an opening auction is not something we check
        return
    }
    if as.IsLiquidityAuction() {
        // auction was started by this engine
        // check if auction expired according to internal logic
        if l.checkSaysStop {
            as.SetReadyToLeave()
        }
        return // we've checked auction we started
    }
    // not a liquidity or opening auction, but it's the end of this auction
    if as.CanLeave() {
        // check internal logic, see if this auction can safely terminate
        if l.auctionNeeded() {
            as.ExtendAuction(params) // extend auction by some parameters based on internal logic
        }
    }
}
```

## Price sub-package

Price subpackage contains the price monitoring engine. It's used to determine if the price movement exceeded the bounds implied by the risk model over a specified horizon at a specified probability level. If that's the case, the price monitoring engine modifies the `AuctionState` to indicate that the price monitoring auction should commence. Once in auction, the engine checks if the auction time has elapsed, and if so if the resulting auction uncrossing price falls within the price monitoring bounds, if that condition is met the `AuctionState` object gets modified to indicate that price monitoring auction should finish, otherwise the  `AuctionState` object gets modified to indicate that the auction should be extended.

Below is the signature of the price monitoring constructor:

```go
func NewMonitor(riskModel RangeProvider, settings types.PriceMonitoringSettings) (*Engine, error)
```

where:

* `RangeProvider` exposes the method `PriceRange(price, yearFraction, probability float64) (float64, float64)` which returns the minimum (`minPrice`) and maximum (`maxPrice`) valid price per current price (`price`), the time horizon expressed as year fraction (`yearFraction`) and probability level (`probability`). `price`, `minPrice`, `maxPrice` are then used to imply `MinMoveDown` and `MaxMoveUp` over `yearFraction` at probability level  `probability` as: `MinMoveDown`=`minPrice` - `price`, `MaxMoveUp`: `maxPrice` - `price`.
* `PriceMonitoringSettings` contains:
  * a list of `yearFraction`, `probability` and `auctionExtension` tuples, where `yearFraction`, `probability` are used in a call to the risk model and `auctionExtension` is the period in seconds by which the current auction should be extended (or initial period of new auction if currently market is in its _"normal"_ trading mode) should the actual price movement over min(`yearFraction`,t), where t is the time since last auction expressed as a year fraction, violate the bounds implied by the `RangeProvider` (`MinMoveDown`, `MaxMoveUp`),
  * `UpdateFrequency` indicates how often `RangeProvider` should be called to update `MinMoveDown`, `MaxMoveUp`.
