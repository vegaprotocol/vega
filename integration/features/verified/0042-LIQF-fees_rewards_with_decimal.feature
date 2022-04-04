Feature: Test decimal places in LP order, liquidity provider reward distribution; Should also cover liquidity-fee-setting and equity-like-share calc and total stake.

Scenario: 001: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades (0042-LIQF-001)
  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
      | USD | 2              |
    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |
    
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | decimal places | position decimal places |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       |
      | USD/DEC19 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       |
      | USD/DEC20 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       |
      | USD/DEC21 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       |
      
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | lp1    | USD   | 100000000000    |
      | lp1    | ETH   | 100000000000000 |
      | party1 | USD   | 10000000000     |
      | party1 | ETH   | 10000000000000  |
      | party2 | USD   | 10000000000     |
      | party2 | ETH   | 10000000000000  |
      
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | USD/DEC19 | 1000000000        | 0.001 | buy  | BID              | 1          | 2000   | submission |
      | lp1 | lp1   | USD/DEC20 | 1000000000        | 0.001 | buy  | BID              | 1          | 200000 | submission  |
      | lp1 | lp1   | USD/DEC21 | 1000000000        | 0.001 | buy  | BID              | 1          | 200000 | submission  |
      | lp1 | lp1   | USD/DEC19 | 1000000000        | 0.001 | buy  | MID              | 2          | 1000   | amendment  |
      | lp1 | lp1   | USD/DEC20 | 1000000000        | 0.001 | buy  | MID              | 2          | 100000 | amendment  |
      | lp1 | lp1   | USD/DEC21 | 1000000000        | 0.001 | buy  | MID              | 2          | 100000 | amendment  |
      | lp1 | lp1   | USD/DEC19 | 1000000000        | 0.001 | sell | ASK              | 1          | 2000   | amendment  |
      | lp1 | lp1   | USD/DEC20 | 1000000000        | 0.001 | sell | ASK              | 1          | 200000 | amendment  |
      | lp1 | lp1   | USD/DEC21 | 1000000000        | 0.001 | sell | ASK              | 1          | 200000 | amendment  |
      | lp1 | lp1   | USD/DEC19 | 1000000000        | 0.001 | sell | MID              | 2          | 1000   | amendment  |
      | lp1 | lp1   | USD/DEC20 | 1000000000        | 0.001 | sell | MID              | 2          | 100000 | amendment  |
      | lp1 | lp1   | USD/DEC21 | 1000000000        | 0.001 | sell | MID              | 2          | 100000 | amendment  |

    Then the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     |
      | party1 | USD/DEC19 | buy  | 1000   | 900000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC20 | buy  | 100000 | 90000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC21 | buy  | 1000   | 90000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC19 | buy  | 10000  | 1000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC20 | buy  | 1000000| 100000000| 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | USD/DEC21 | buy  | 10000  | 100000000| 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC19 | sell | 1000   | 1100000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC20 | sell | 100000 | 110000000| 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC21 | sell | 1000   | 110000000| 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC19 | sell | 10000  | 1000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC20 | sell | 1000000| 100000000| 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | USD/DEC21 | sell | 10000  | 100000000| 0                | TYPE_LIMIT | TIF_GTC |

     Then the opening auction period ends for market "USD/DEC19"
     Then the opening auction period ends for market "USD/DEC20"
     Then the opening auction period ends for market "USD/DEC21"

    And the market data for the market "USD/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000000    | TRADING_MODE_CONTINUOUS | 100000  | 863654    | 1154208   | 3556900000   | 1000000000     | 10000         |

    And the market data for the market "USD/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000000  | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3556900000   | 0              | 1000000       |

    And the market data for the market "USD/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000000  | TRADING_MODE_CONTINUOUS | 100000  | 86365368  | 115420826 | 3556900000   | 0              | 10000         |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    # max_oi: max open interest

    Then the order book should have the following volumes for market "USD/DEC19":
      | side | price    | volume   |
      | buy  | 898000   | 3000     |
      | buy  | 900000   | 11000    |
      | buy  | 999000   | 5000     |
      | sell | 1102000  | 3000     |
      | sell | 1100000  | 11000    |
      | sell | 1001000  | 5000     |

    Then the order book should have the following volumes for market "USD/DEC20":
      | side | price      | volume   |
      | buy  | 89800000   | 0     |
      | buy  | 90000000   | 100000    |
      | buy  | 99900000   | 0     |
      | sell | 110200000  | 0     |
      | sell | 110000000  | 100000    |
      | sell | 100100000  | 0     |

    # Then the order book should have the following volumes for market "USD/DEC21":
    #   | side | price    | volume   |
    #   | buy  | 89800000   | 3000     |
    #   | buy  | 90000000   | 11000    |
    #   | buy  | 99900000   | 5000     |
    #   | sell | 110200000  | 3000     |
    #   | sell | 110000000  | 11000    |
    #   | sell | 100100000  | 5000     |

    And the liquidity provider fee shares for the market "USD/DEC19" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 1000000000              |

    # And the liquidity provider fee shares for the market "USD/DEC20" should be:
    #   | party | equity like share | average entry valuation |
    #   | lp1   | 1                 | 100000000000              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin       | general        | bond       |
      | lp1    | ETH   | USD/DEC19 | 10243882344  | 99988756117656 | 1000000000 |
      | party1 | USD   | USD/DEC19 | 1176961234   | 10000000000    | 0          |
      | party2 | USD   | USD/DEC19 | 4815112741   | 10000000000    | 0          |
      
  Scenario: 002, no decimal

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          |
      | ETH/MAR22 | USD        | USD   | log-normal-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring-1 | default-eth-for-future |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |

    Given the average block duration is "2"

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100000  | 864       | 1154      | 35569         | 10000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    # max_oi: max open interest

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 8      |
      | buy  | 900   | 1      |
      | buy  | 999   | 14     |
      | sell | 1102  | 7      |
      | sell | 1100  | 1      |
      | sell | 1001  | 14     |

  
