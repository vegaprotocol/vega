# Positions Engine

Product Specification: [product/specs/0012-position-resoluton.md](https://gitlab.com/vega-protocol/product/blob/master/specs/0012-position-resoluton.md)

The Positions Engine maintains a map of `partyID` to `MarketPosition struct`. In
the struct are:

* `size`: the actual volume (orders having been accepted)
* `buy` and `sell` : volume of buy and sell orders not yet accepted

For tracking actual and potential volume:

* `RegisterOrder`, called in `SubmitOrder` and `AmendOrder`, adds to the `buy`
  xor `sell` potential volume
* `UnregisterOrder`, called in `AmendOrder` and `CancelOrder`, subtracts from
  the `buy` xor `sell` potential volume
* `Update` deals with an accepted order and does the following:
  * transfers actual volume from the seller to the buyer:
    ```go
    buyer.size += int64(trade.Size)  // increase
    seller.size -= int64(trade.Size) // decrease
    ```
  * decreases potential volume for both the buyer and seller:
    ```go
    buyer.buy -= int64(trade.Size)   // decrease
    seller.sell -= int64(trade.Size) // decrease
    ```

The Position for a party is updated before an order is accepted/rejected.

The Risk Engine determines if the order is acceptable.

Settlement is done pre-trade on the old position, because a trader is liable for
their position, and also on the new position.
