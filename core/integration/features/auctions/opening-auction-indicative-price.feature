Feature: Set up a market, create indiciative price different to actual opening auction uncross price

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 8                | default-none | default-basic    | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 8     |
  #| network.floatingPointUpdates.delay | 30s   |

  @IPOTest
  Scenario: Simple test with different indicative price before auction uncross
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | party5 | BTC   | 100000000 |
      | party6 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    When the network moves ahead "3" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    # start with submitting trades that produce an indicative uncrossing price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party6 | ETH/DEC19 | buy  | 100    | 10900 | 0                | TYPE_LIMIT | TIF_GFA | t6-b-1    |
      | party5 | ETH/DEC19 | sell | 100    | 10900 | 0                | TYPE_LIMIT | TIF_GFA | t5-s-1    |
    # continue opening auction
    When the network moves ahead "4" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the parties cancel the following orders:
      | party  | reference |
      | party6 | t6-b-1    |
      | party5 | t5-s-1    |
    # place orders to set the actual price point at which we'll uncross to be 10000
    # When the network moves ahead "1" blocks
    # Then the market data for the market "ETH/DEC19" should be:
    #   | trading mode                 |
    #   | TRADING_MODE_OPENING_AUCTION |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | party1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | party2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 60000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 60000             | 0.1 | sell | MID              | 50         | 100    | submission |

    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode            |
      | TRADING_MODE_CONTINUOUS |
    ## We're seeing these events twice for some reason
    Then debug trades
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 3    | party2 |
      | party1 | 10000 | 2    | party2 |
      | party1 | 10000 | 3    | party2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    ## Network for distressed party1 -> cancelled, nothing on the book is remaining
    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party2 | t2-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_CANCELLED |
      | party2 | t2-s-2    | STATUS_CANCELLED |
      | party1 | t1-b-3    | STATUS_CANCELLED |
      | party2 | t2-s-3    | STATUS_FILLED    |
      | party5 | t5-s-1    | STATUS_CANCELLED |
      | party6 | t6-b-1    | STATUS_CANCELLED |
    #| party6 | t6-b-1    | STATUS_FILLED    |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | ref price |
      | 10000      | TRADING_MODE_CONTINUOUS | 5       | 9985      | 10015     | 10000     |


