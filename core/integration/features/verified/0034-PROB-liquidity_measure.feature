Feature: Tests confirming probability of trading acceptance criteria (0038-OLIQ-001, 0038-OLIQ-002, 0009-MRKP-002, 0009-MRKP-006, 0018-RSKM-007, 0018-RSKM-008)

  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | market.liquidityProvision.shapes.maxSize            | 10    |
      | network.markPriceUpdateMaximumFrequency             | 0s    |

  Scenario: 001: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size(0034-PROB-001);Probability of trading decreases away from the mid-price (0034-PROB-005). Tested with varying decimal places.

    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00000190128526884174 | 0  | 0.016 | 2.5   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999     | 300               |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 3              |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2              | 2                       | 1e6                    | 1e6                       |

    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount                |
      | party1     | ETH   | 1000000000000         |
      | party2     | ETH   | 1000000000            |
      | party-lp-1 | ETH   | 100000000000000000000 |
      | party3     | ETH   | 100000000000          |

    And the parties submit the following liquidity provision:
      | id  | party      | market id | commitment amount | fee | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 100000 | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 90000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 80000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 70000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 60000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 50000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 40000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 30000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | buy  | BID              | 1          | 20000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 20000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 30000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 40000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 50000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 60000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 70000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 80000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 90000  | lp-1-ref  | amendment  |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | 0.1 | sell | ASK              | 1          | 100000 | lp-1-ref  | amendment  |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 1199990000 | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 1200010000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 1200000000 | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 1200000000 | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1200000000" for the market "ETH/DEC19"

    Then the liquidity provisions should have the following states:
      | id  | party      | market    | commitment amount | status        |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000000     | STATUS_ACTIVE |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price      | volume |
      | buy  | 1199990000 | 1      |
      | buy  | 1199970000 | 926    |
      | buy  | 1199960000 | 926    |
      | buy  | 1199950000 | 926    |
      | buy  | 1199940000 | 926    |
      | buy  | 1199930000 | 926    |
      | buy  | 1199920000 | 926    |
      | buy  | 1199910000 | 926    |
      | buy  | 1199900000 | 927    |
      | sell | 1200010000 | 1      |
      | sell | 1200030000 | 926    |
      | sell | 1200040000 | 926    |
      | sell | 1200050000 | 926    |
      | sell | 1200060000 | 926    |
      | sell | 1200070000 | 926    |
      | sell | 1200080000 | 926    |
      | sell | 1200090000 | 926    |
      | sell | 1200100000 | 926    |

  Scenario: 004: LP Volume being pushed by limit of Probability of Trading (capped at 1e-8)
    #Price Monitoring has been removed as Prob in Price Monitoring only take up to 15 decimal places which will prevent scenatio which will trigger the ProbOfTrading cap at 1e-8

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau     | mu | r | sigma |
      | 0.000001      | 0.00273 | 0  | 0 | 1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |

    And the markets:
      | id         | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount              |
      | lp1    | ETH2  | 1000000000000000000 |
      | party1 | ETH2  | 10000000            |
      | party2 | ETH2  | 10000000            |

    And the parties submit the following liquidity provision:
      | id          | party | market id  | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 600        | 600    | submission |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 600        | 600    | amendment  |

    And the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH2/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH2/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH2/MAR22 | sell | 1      | 1109  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH2/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH2/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3611         | 50000000       | 10            |

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    # ProbOfTrading is floored at 1e-8 when LP pegged ref offset from 500 onward, we use 600 in this test case

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume |
      | buy  | 300   | 166667 |
      | buy  | 900   | 1      |
      | sell | 1109  | 1      |
      | sell | 1709  | 29257  |

  Scenario: 005: Create LP shape that pegs to mid and deploys volumes and price between best ask and best bid (0034-PROB-005)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |

    And the markets:
      | id         | quote name | asset | risk model                | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH2/MAR22 | ETH2       | ETH2  | default-simple-risk-model | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount              |
      | lp1    | ETH2  | 1000000000000000000 |
      | party1 | ETH2  | 10000000            |
      | party2 | ETH2  | 10000000            |

    And the parties submit the following liquidity provision:
      | id          | party | market id  | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | MID              | 20         | 5      | submission |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | MID              | 30         | 10     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | MID              | 40         | 15     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | MID              | 5          | 20     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | MID              | 5          | 20     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | MID              | 40         | 15     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | MID              | 30         | 10     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | MID              | 20         | 5      | amendment  |

    And the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH2/MAR22 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH2/MAR22 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH2/MAR22 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH2/MAR22 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH2/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "100"

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    # checking the pegged price and pegged volume
    #volume = ceiling(liquidity_obligation x liquidity-normalised-proportion / probability_of_trading / price)

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume |
      | buy  | 80    | 32895  |
      | buy  | 85    | 247679 |
      | buy  | 90    | 175439 |
      | buy  | 95    | 110804 |
      | sell | 105   | 100251 |
      | sell | 110   | 143541 |
      | sell | 115   | 183067 |
      | sell | 120   | 21930  |

