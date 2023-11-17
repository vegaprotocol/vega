Feature: stop orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC20 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future | 1e-3                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  Scenario: A linked stop order will change it's order size based on the linked order

    # setup accounts
    Given time is updated to "2019-11-30T00:00:00Z"
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 100000   |
      | party2 | BTC   | 100000   |
      | party3 | BTC   | 100000   |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | aux3   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 10     |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 10     |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setup party1 position, open a 10 long position
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1| ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sellorder |
      | party2| ETH/DEC19 | buy  | 11     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buyorder  |


    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | ra price trigger | reference | ra size override setting | ra size override reference |
      | party1| ETH/DEC19 | buy  | 10     |  0    | 0                | TYPE_MARKET| TIF_IOC | reduce | 51               | stop1     | ORDER                    | buyorder                   |

    Then the stop orders should have the following states
      | party  | market id | status          | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING  | stop1     |

    # move the price down to trigger the stop order
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party3| ETH/DEC19 | buy  | 1      | 52    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2| ETH/DEC19 | sell | 1      | 52    | 1                | TYPE_LIMIT | TIF_GTC |

    # Stop order should have triggered
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | stop1     |
