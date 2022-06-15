Feature: Test when the "market.liquidity.probabilityOfTrading.tau.scaling" is applied, whether its impacted with the delay parameter "network.floatingPointUpdates.delay"

  # Related spec files:
  #  ../spec/0032-price-monitoring.md

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0     |
      | network.floatingPointUpdates.delay            | 5s    |
      | market.auction.minimumDuration                | 10    |
 
    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1000    | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          |
      | ETH/DEC21 | ETH        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 10               | fees-config-1 | price-monitoring-1 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

  Scenario: set tau_scaling parameter
    Given the following network parameters are set:
      | name                                             | value |
      |market.liquidity.probabilityOfTrading.tau.scaling |   1   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 40000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 40000              | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 972       | 1029      | 743          | 40000           | 10            |
    # set time
     Then time is updated to "2019-11-30T00:10:05Z"
     And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 10     |
      | buy  | 990   | 1      |
      | buy  | 999   | 81     |
      | sell | 1100  | 10     |
      | sell | 1010  | 1      |
      | sell | 1001  | 80     |
    
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 1020  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1000    | 972       | 1029      | 818          | 40000           | 11            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 10     |
      | buy  | 990   | 1      |
      | buy  | 999   | 81     |
      | sell | 1100  | 10     |
      | sell | 1010  | 1      |
      | sell | 1001  | 80     |
      | sell | 1020  | 20     |
  
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party0 | ETH   | ETH/DEC21 | 7245     | 999952760 |

    Then the following network parameters are set:
      | name                                             | value |
      |market.liquidity.probabilityOfTrading.tau.scaling |   10  |

    # set time
    Then time is updated to "2019-12-30T00:11:05Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 1020  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1000    | 973       | 1030      | 892          | 40000           | 12            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party0 | ETH   | ETH/DEC21 | 7357     | 999952655 |

     And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 10     |
      | buy  | 990   | 1      |
      | buy  | 999   | 81     |
      | sell | 1100  | 10     |
      | sell | 1010  | 1      |
      | sell | 1001  | 80     |
      | sell | 1020  | 40     |
