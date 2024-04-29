Feature: Set up a market, create indiciative price different to actual opening auction uncross price

  Background:
    Given the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 2           | -3            | 0.2                    |
    Given the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | my-simple-risk-model | default-margin-calculator | 5                | default-none | default-basic    | default-eth-for-future | 0.25                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 5     |
      | market.auction.maximumDuration          | 10s   |
      | network.floatingPointUpdates.delay      | 10s   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  @DebugAuctionMax @Expires
  Scenario: 0043-MKTL-012 Simple test verifying the market is cancelled if it failes to leave opening auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | party5 | BTC   | 100000000 |
      | party6 | BTC   | 100000000 |
      | party7 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # Start market with some dead time
    And the network moves ahead "5" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    # Ensure an indicative price/volume of 10, although we will not uncross at this price point
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party6 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GFA | t6-b-1    |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GFA | t5-s-1    |
    # place orders to set the actual price point at which we'll uncross to be 10000
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    When the network moves ahead "10" blocks

    # Now the market should be cancelled
    Then the last market state should be "STATE_CANCELLED" for the market "ETH/DEC19"
