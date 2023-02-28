Feature: Test liquidity monitoring

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 1s    |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 1     |
      | network.floatingPointUpdates.delay            | 10s   |
      | market.auction.minimumDuration                | 1     |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
    And the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.0           | 1.0            | 2              |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99        | 300               |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1     | margin-calculator-1       | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | lprov1 | ETH   | 1000000000 |
      | lprov2 | ETH   | 1000000000 |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | lprov1 | USD   | 500000    |
      | lprov2 | USD   | 500000    |

  Scenario: 001: A market which enters a state requiring liquidity auction through increased open interest during a block but then leaves state again prior to block completion never enters liquidity auction. (0035-LIQM-005)
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon |min bound| max bound|  target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     |990      |1010      |   1000        | 1000           |  0            |

    Then the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | horizon |min bound| max bound|  target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 100     |990      |1010      |   1000        | 1000           |  0            |
  
    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 988   | 0      |
      | buy  | 990   | 0      |
      | buy  | 1000  | 0      |
      | sell | 1000  | 0      |
      | sell | 1010  | 0      |
      | sell | 1012  | 0      |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0     | 0               | -100000000   |
      | party2 | 0     | 0               | -100000000   |
      | lprov1 | 0     | 0               | 0            |

    
 
