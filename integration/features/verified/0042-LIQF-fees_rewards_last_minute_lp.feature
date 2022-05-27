Feature: Test if a LP comes in the last second before LP fee gets distributed.liquidity provider reward distribution; 
  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |

    Given the average block duration is "2"

  Scenario: 1 LP joining at start, and trade for fee purpose happends; then 2nd LP joins later
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
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

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 75     |
      | buy  | 900   | 1      |
      | buy  | 1000  | 0      |
      | sell | 1000  | 15     |
      | sell | 1001  | 0      |
      | sell | 1102  | 0      |
      | sell | 1100  | 1      |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    Then time is updated to "2019-11-30T00:10:04Z"
    # lp2 is coming to "steal" the lp fee!!!!!!!!
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 100000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 100000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.5361565935340229 | 10000                   |
      | lp2   | 0.4638434064659771 | 115590                  |

    Then time is updated to "2019-11-30T00:10:05Z"
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 10     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 10     | USD   |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 1                 | 0.001 | buy  | BID              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 1                 | 0.001 | sell | ASK              | 1          | 2      | amendment  |

    Then time is updated to "2019-11-30T00:10:06Z"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.9999913488074331 | 10000                   |
      | lp2   | 0.0000086511925669 | 115590                  |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    
