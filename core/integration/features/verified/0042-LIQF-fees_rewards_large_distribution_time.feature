Feature: Test liquidity provider reward distribution; Check what happens when distribution period is large(both in genesis)

  Background:

    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    And the following network parameters are set:
      | name                                                | value  |
      | market.value.windowLength                           | 1h     |
      | market.stake.target.timeWindow                      | 24h    |
      | market.stake.target.scalingFactor                   | 1      |
      | market.liquidity.targetstake.triggering.ratio       | 0      |
      | market.liquidity.providers.fee.distributionTimeStep | 720h   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |

    Given the average block duration is "2"

  Scenario: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades (0042-LIQF-006)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment  |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 1000         | 10000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    # max_oi: max open interest

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 75     |
      | buy  | 900   | 1      |
      | buy  | 999   | 14     |
      | sell | 1102  | 61     |
      | sell | 1100  | 1      |
      | sell | 1001  | 14     |

    #volume = ceiling(liquidity_obligation x liquidity-normalised-proportion / probability_of_trading / price)
    #for any price better than the bid price or better than the ask price it returns 0.5
    #for any price in within 500 price ticks from the best bid/ask (i.e. worse than) it returns the probability as returned by the risk model (in this case 0.1 scaled by 0.5.
    #priceLvel at 898:10000*(1/3)/0.05/898=74.23
    #priceLvel at 999:10000*(2/3)/0.5/999=13.34
    #priceLvel at 1102:10000*(1/3)/0.05/1102=60.49
    #priceLvel at 1001:10000*(2/3)/0.5/1001=13.32

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 10000                   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 10680  | 999979320 | 10000 |
      | party1 | USD   | ETH/MAR22 | 2758   | 99997242  | 0     |
      | party2 | USD   | ETH/MAR22 | 2652   | 99997348  | 0     |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 11787  | 999977484 | 10000 |
      | party1 | USD   | ETH/MAR22 | 1800   | 99998202  | 0     |
      | party2 | USD   | ETH/MAR22 | 1812   | 99998875  | 0     |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 75     |
      | buy  | 900   | 1      |
      | buy  | 1000  | 0      |
      | sell | 1000  | 15     |
      | sell | 1001  | 0      |
      | sell | 1102  | 0      |
      | sell | 1100  | 1      |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general   |
      #| lp1   | USD   | ETH/MAR22 | 12522  | 999976749 |
      | lp1   | USD   | ETH/MAR22 | 11787  | 999977484 |


    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

   # lp fee got cumulated since the distribution period is large
    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    Then time is updated to "2019-11-30T00:20:05Z"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    Then debug orders

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 15   | lp1    |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general   |
      | lp1   | USD   | ETH/MAR22 | 14381  | 999975631 |
      #| lp1   | USD   | ETH/MAR22 | 13257  | 999976755 |

    # lp fee got cumulated since the distribution period is large
    And the accumulated liquidity fees should be "35" for the market "ETH/MAR22"

    # lp fee got paid to lp1 when time is over the "fee.distributionTimeStep"
    Then time is updated to "2024-12-30T00:30:05Z"

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 35     | USD   |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general   |
      | lp1   | USD   | ETH/MAR22 | 14381  | 999975666 |
      #| lp1   | USD   | ETH/MAR22 | 13257  | 999976790 |
