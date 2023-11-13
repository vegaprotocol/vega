Feature: Tests impact from change of tau.scaling parameter on probability of trading

  Scenario: 001: set tau.scaling to 1
    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1     |
      | limits.markets.maxPeggedOrders                    | 4     |
      | market.liquidity.providersFeeCalculationTimeStep  | 600s  |
    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.0              | 24h         | 1.0            |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 100         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 3572      | 1                    | buy  | BID              | 3572   | 200    |
      | lp1   | ETH/MAR22 | 2778      | 1                    | buy  | MID              | 2777   | 100    |
      | lp1   | ETH/MAR22 | 1924      | 1                    | sell | ASK              | 1924   | 200    |
      | lp1   | ETH/MAR22 | 2273      | 1                    | sell | MID              | 2273   | 100    |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/MAR22 | 2784      | 1                    | buy  | BID              | 2784   | 2      |
      | lp2   | ETH/MAR22 | 2502      | 1                    | buy  | MID              | 2502   | 1      |
      | lp2   | ETH/MAR22 | 2269      | 1                    | sell | ASK              | 2269   | 2      |
      | lp2   | ETH/MAR22 | 2498      | 1                    | sell | MID              | 2498   | 1      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100000  | 864       | 1154      | 1012920      | 1000000000     | 60            |



    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 700   | 3572   |
      | buy  | 864   | 0      |
      | buy  | 898   | 2784   |
      | buy  | 900   | 2778   |
      | buy  | 999   | 2502   |
      | sell | 1001  | 2498   |
      | sell | 1100  | 2274   |
      | sell | 1102  | 2269   |
      | sell | 1154  | 0      |
      | sell | 1300  | 1924   |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 500000000               |
      | lp2   | 0.5               | 1000000000              |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 60     | -294000        | 0            |
      | party2 | -40    | 196000         | 98000        |
      | lp1    | 0      | 0              | 0            |
      | lp2    | -20    | 0              | 0            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1902" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 573    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 1328   | USD   |

  Scenario: 002: set tau.scaling to 10
    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 10    |
      | limits.markets.maxPeggedOrders                    | 4     |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 200    |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 100    |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 200    |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 100    |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 1      |
    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 500000000               |
      | lp2   | 0.5               | 1000000000              |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 41     | 0              | 0            |
      | party2 | -40    | 0              | 4900         |
      | lp1    | 0      | 0              | 0            |
      | lp2    | -1     | -4900          | 0            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1996" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 610    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 1385   | USD   |

  Scenario: 003: set tau.scaling to 1000
    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1000  |
      | limits.markets.maxPeggedOrders                    | 4     |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 3572      | 1                    | buy  | BID              | 3572   | 200    |
      | lp1   | ETH/MAR22 | 2778      | 1                    | buy  | MID              | 2777   | 100    |
      | lp1   | ETH/MAR22 | 1924      | 1                    | sell | ASK              | 1924   | 200    |
      | lp1   | ETH/MAR22 | 2273      | 1                    | sell | MID              | 2273   | 100    |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/MAR22 | 2784      | 1                    | buy  | BID              | 2784   | 2      |
      | lp2   | ETH/MAR22 | 2502      | 1                    | buy  | MID              | 2502   | 1      |
      | lp2   | ETH/MAR22 | 2269      | 1                    | sell | ASK              | 2269   | 2      |
      | lp2   | ETH/MAR22 | 2498      | 1                    | sell | MID              | 2498   | 1      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 500000000               |
      | lp2   | 0.5               | 1000000000              |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 60     | -294000        | 0            |
      | party2 | -40    | 196000         | 98000        |
      | lp1    | 0      | 0              | 0            |
      | lp2    | -20    | 0              | 0            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1902" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 572    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 1329   | USD   |

  Scenario: 004: set tau.scaling to 1, smaller lp offset
    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1     |
      | limits.markets.maxPeggedOrders                    | 8     |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 20     |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 10     |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 20     |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 10     |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 1      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 500000000               |
      | lp2   | 0.5               | 1000000000              |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 42     | 0              | 0            |
      | party2 | -40    | 0              | 8900         |
      | lp1    | -1     | -4000          | 0            |
      | lp2    | -1     | -4900          | 0            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1992" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 903    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 1088   | USD   |

  Scenario: 005: set tau.scaling to 10, smaller lp offset
    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 10    |
      | limits.markets.maxPeggedOrders                    | 4     |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 20     |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 10     |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 20     |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 10     |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 1      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 500000000               |
      | lp2   | 0.5               | 1000000000              |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 42     | 0              | 0            |
      | party2 | -40    | 0              | 8900         |
      | lp1    | -1     | -4000          | 0            |
      | lp2    | -1     | -4900          | 0            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1992" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 969    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 1022   | USD   |

  Scenario: 006: set tau.scaling to 1000, smaller lp offset
    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1000  |
      | limits.markets.maxPeggedOrders                    | 4     |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 20     |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 10     |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 20     |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 10     |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 1      | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | MID              | 1      | 1      |
 
    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 500000000               |
      | lp2   | 0.5               | 1000000000              |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 42     | 0              | 0            |
      | party2 | -40    | 0              | 8900         |
      | lp1    | -1     | -4000          | 0            |
      | lp2    | -1     | -4900          | 0            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1992" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 992    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 999    | USD   |


  Scenario: market.liquidity.probabilityOfTrading.tau.scaling updated and commitment competiveness reevaluated (0034-PROB-004)

    Given the following network parameters are set:
      | name                                              | value |
      | market.value.windowLength                         | 1h    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1     |
      | limits.markets.maxPeggedOrders                    | 4     |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |
    And the average block duration is "1"
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | ETH        | USD   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.001                  | 0                         | SLA        |

    # Initalise the market with 2 LPs with equal commitments but different competiveness
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 1000000           | 0.001 | submission |
      | lp2 | lp2   | ETH/MAR22 | 1000000           | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2000      | 1000                 | buy  | BID              | 2000   | 1      |
      | lp1   | ETH/MAR22 | 2000      | 1000                 | sell | ASK              | 2000   | 1      |
      | lp2   | ETH/MAR22 | 2000      | 1000                 | buy  | BID              | 2000   | 100    |
      | lp2   | ETH/MAR22 | 2000      | 1000                 | sell | ASK              | 2000   | 100    |
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/MAR22"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"

    # - Check the 'quality' of the supplied liquidity by checking the average score at the end of the epoch.
    # - Update the network parameter, then check the 'quality' of the supplied liquidity has chnaged.
    Given the network moves ahead "1" epochs
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | average score |
      | lp1   | 0.5               | 1000000                 | 0.6249997841  |
      | lp2   | 0.5               | 2000000                 | 0.3750002159  |
    When the following network parameters are set:
      | name                                              | value |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1000  |
    And the network moves ahead "1" epochs
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | average score |
      | lp1   | 0.5               | 1000000                 | 0.5958598141  |
      | lp2   | 0.5               | 2000000                 | 0.4041401859  |



