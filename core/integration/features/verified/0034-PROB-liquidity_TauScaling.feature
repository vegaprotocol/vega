Feature: Tests impact from change of tau.scaling parameter on probability of trading, 0038-OLIQ-006

  Scenario: 001: set tau.scaling to 1
  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling   | 1     |

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

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 100            | 0.001                  | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | BID              | 1          | 200    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | MID              | 1          | 100    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | ASK              | 1          | 200    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | MID              | 1          | 100    | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | MID              | 1          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | MID              | 1          | 1      | submission |

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 295    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1606   | USD   |

  Scenario: 002: set tau.scaling to 10
  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling   | 10    |
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

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 0.001                  | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | BID              | 1          | 200    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | MID              | 1          | 100    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | ASK              | 1          | 200    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | MID              | 1          | 100    | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | MID              | 1          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | MID              | 1          | 1      | submission |

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 291    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1610   | USD   |

  Scenario: 003: set tau.scaling to 1000
  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling   | 1000  |
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

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 0.001                  | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | BID              | 1          | 200    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | MID              | 1          | 100    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | ASK              | 1          | 200    | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | MID              | 1          | 100    | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | MID              | 1          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | MID              | 1          | 1      | submission |

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 289    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1612   | USD   |

  Scenario: 004: set tau.scaling to 1, smaller lp offset
  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling   | 1     |

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


    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 0.001                  | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | BID              | 1          | 20     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | MID              | 1          | 10     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | ASK              | 1          | 20     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | MID              | 1          | 10     | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | MID              | 1          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | MID              | 1          | 1      | submission |

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 815    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1086   | USD   |

  Scenario: 005: set tau.scaling to 10, smaller lp offset
  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling   | 10    |

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


    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 0.001                  | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | BID              | 1          | 20     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | MID              | 1          | 10     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | ASK              | 1          | 20     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | MID              | 1          | 10     | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | MID              | 1          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | MID              | 1          | 1      | submission |

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 910    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 991    | USD   |

  Scenario: 006: set tau.scaling to 1000, smaller lp offset
  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.probabilityOfTrading.tau.scaling   | 1000  |

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

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 0.001                  | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | USD   | 100000000000 |
      | lp2    | USD   | 100000000000 |
      | lp3    | USD   | 100000000000 |
      | party1 | USD   | 10000000000  |
      | party2 | USD   | 10000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | BID              | 1          | 20     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | buy  | MID              | 1          | 10     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | ASK              | 1          | 20     | submission |
      | lp1 | lp1   | ETH/MAR22 | 500000000         | 0.001 | sell | MID              | 1          | 10     | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | buy  | MID              | 1          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 500000000         | 0.002 | sell | MID              | 1          | 1      | submission |

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 946    | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 955    | USD   |

