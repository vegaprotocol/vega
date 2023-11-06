Feature: Test interactions between different auction types (0035-LIQM-001)

  # Related spec files:
  #  ../spec/0026-auctions.md
  #  ../spec/0032-price-monitoring.md
  #  ../spec/0035-liquidity-monitoring.md

  Background:
    Given the following network parameters are set:
      | name                                               | value |
      | network.floatingPointUpdates.delay                 | 10s   |
      | market.auction.minimumDuration                     | 10    |
      | network.markPriceUpdateMaximumFrequency            | 0s    |
      | market.liquidity.successorLaunchWindowLength       | 1s    |
      | limits.markets.maxPeggedOrders                     | 4     |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.0              | 24h         | 1.0            |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99        | 300               |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 2       | 0.999995    | 200               |
      | 1       | 0.999999    | 300               |
    And the markets:
      | id        | quote name | asset | liquidity monitoring    | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH        | ETH   | lqm-params              | simple-risk-model-1     | default-margin-calculator | 10               | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         | default-futures |
      | ETH/DEC22 | ETH        | ETH   | lqm-params              | log-normal-risk-model-1 | default-margin-calculator | 10               | fees-config-1 | price-monitoring-2 | default-eth-for-future | 0.5                    | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | lpprov | ETH   | 1000000000 |

  Scenario: Assure minimum auction length is respected
    Given the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 400   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | submission |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | submission |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | submission |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | MID              | 2      | 1      |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | MID              | 2      | 1      |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 4000           | 10            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # try to end price auction (gets extended by min auction length as max(400,300) = 400)
    When the network moves ahead "350" blocks

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    When the network moves ahead "50" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario: When trying to exit opening auction liquidity monitoring doesn't get triggered, hence the opening auction uncrosses and market goes into continuous trading mode (0026-AUCT-004)

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | submission |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 500    | 10     |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 500    | 10     |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"

    # target_stake = mark_price x max_oi x target_stake_scaling_factor x max(risk_factor_long, risk_factor_short) = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 10000          | 10            |
