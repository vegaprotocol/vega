Feature: Oracle price data within range is used to determine the mid price

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
      | market.auction.minimumDuration          | 1     |
      | limits.markets.maxPeggedOrders          | 2     |
    Given the following assets are registered:
      | id  | decimal places |
      | DAI | 2              |
    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 20s         | 10             |
    And the log normal risk model named "dai-lognormal-risk":
      | risk aversion | tau         | mu | r | sigma |
      | 0.00001       | 0.000114077 | 0  | 0 | 0.41  |
    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | price1.USD.value | TYPE_INTEGER | 0              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model         | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      | max price cap | binary | fully collateralised | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 |
      | DAI/DEC22 | DAI        | DAI   | lqm-params           | dai-lognormal-risk | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2              | 0.25                   | 0                         | default-futures | 4500000       | false  | false                | weight     | 1            | 1           | 0           | 0,0,1,0        | 0s,0s,10s,0s               | oracle1 |

  @MidPrice @NoPerp @Capped
  Scenario: 0016-PFUT-016: When a market is setup to use oracle based mark price and the value received from oracle is less than max_price then it gets used as is and mark-to-market flows are calculated according to that price.
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | DAI   | 110000000 |
      | party2 | DAI   | 110000000 |
      | party3 | DAI   | 110000000 |
      | party4 | DAI   | 110000000 |
      | party5 | DAI   | 110000000 |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 200000            | 0.01 | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 200000            | 0.01 | lp-1      | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party1 | DAI/DEC22 | 5         | 3                    | buy  | MID              | 5      | 100000 |
      | party1 | DAI/DEC22 | 5         | 3                    | sell | MID              | 5      | 100000 |

    #0016-PFUT-014:When `max_price` is specified, an order with a `price > max_price` gets rejected.
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference | error               |
      | party2 | DAI/DEC22 | buy  | 1      | 2500000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |                     |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |                     |
      | party3 | DAI/DEC22 | sell | 1      | 3500000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |                     |
      | party3 | DAI/DEC22 | sell | 1      | 4499999 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |                     |
      | party3 | DAI/DEC22 | sell | 1      | 8000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  | invalid order price |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price   | size | seller |
      | party2 | 3500000 | 1    | party3 |

    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000    | 2500000               | 3499999          | 4499999                 |
    And debug detailed orderbook volumes for market "DAI/DEC22"
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price   | volume |
      | sell | 3599999 | 5      |
      | sell | 4499999 | 1      |
      | buy  | 3400000 | 5      |
      | buy  | 2500000 | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party4 | DAI/DEC22 | buy  | 1      | 3200000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party5 | DAI/DEC22 | sell | 1      | 3200000 | 1                | TYPE_LIMIT | TIF_GTC | party3-1  |
    When the network moves ahead "2" blocks
    Then the mark price should be "3500000" for the market "DAI/DEC22"

    #0016-PFUT-017: When `max_price` set by oracle, `mark price > max_price`, then it gets ignored and mark-to-market settlement doesn't occur
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value  | time offset |
      | price1.USD.value | 330000 | -1s         |

    When the network moves ahead "2" blocks
    Then the mark price should be "3500000" for the market "DAI/DEC22"

    #0016-PFUT-016: When `max_price` set by oracle, `mark price < max_price`, then it gets used
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | price1.USD.value | 32000 | -1s         |

    When the network moves ahead "2" blocks
    Then the mark price should be "3200000" for the market "DAI/DEC22"

    # #MTM happens if mark price < max_price
    And the following transfers should happen:
      | type                   | from   | to     | from account            | to account              | market id | amount | asset |
      | TRANSFER_TYPE_MTM_LOSS | party1 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | DAI/DEC22 | 300000 | DAI   |
      | TRANSFER_TYPE_MTM_WIN  | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | DAI/DEC22 | 300000 | DAI   |


