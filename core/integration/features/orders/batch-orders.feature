Feature: Iceberg orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                      | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |

  @batch
  Scenario: Batch with normal orders and icebergs
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 1000000000    |
      | aux    | BTC   | 1000000 |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id                                                        | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following iceberg orders:
      | party  | market id                                                        | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party1 | ETH/DEC19 | sell | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 4            | 5            |
      | party2 | ETH/DEC19 | sell | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 4            | 5            |

    Then the party "party3" starts a batch instruction

    Then the party "party3" adds the following orders to a batch:
      | market id | side | volume | price | type       | tif     | reference |
      | ETH/DEC19 | buy  | 4      | 2     | TYPE_LIMIT | TIF_GTC | party3    |


    Then the party "party3" adds the following iceberg orders to a batch:
      | market id                                                        | side | volume | price | type       | tif     | reference    | peak size | minimum visible size |
      | ETH/DEC19 | buy | 3    | 2     | TYPE_LIMIT | TIF_GTC | this-order-1 | 2            | 1            |
      | ETH/DEC19 | buy | 2    | 2     | TYPE_LIMIT | TIF_GTC | this-order-2 | 2            | 1            |

    Then the party "party3" submits their batch instruction

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 2     | 4    |
      | party3 | party2 | 2     | 3    | #the iceberg of party1 will refresh and lose priority so the next trade will be with party2
      | party3 | party1 | 2     | 2    |
