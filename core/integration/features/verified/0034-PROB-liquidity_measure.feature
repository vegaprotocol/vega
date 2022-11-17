Feature: Tests confirming probability of trading acceptance criteria (0038-OLIQ-001, 0038-OLIQ-002, 0009-MRKP-001, 0009-MRKP-002, 0009-MRKP-006, 0018-RSKM-007, 0018-RSKM-008)

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

  Scenario: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size(0034-PROB-001);Probability of trading decreases away from the mid-price (0034-PROB-005). Tested with varying decimal places.

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
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | position decimal places |
      | ETH/DEC19 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2              | 2                       |

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
      | buy  | 1199970000 | 1859   |
      | buy  | 1199960000 | 1863   |
      | buy  | 1199950000 | 1867   |
      | buy  | 1199940000 | 1870   |
      | buy  | 1199930000 | 1874   |
      | buy  | 1199920000 | 1877   |
      | buy  | 1199910000 | 1881   |
      | buy  | 1199900000 | 1885   |
      | sell | 1200010000 | 1      |
      | sell | 1200030000 | 1859   |
      | sell | 1200040000 | 1863   |
      | sell | 1200050000 | 1866   |
      | sell | 1200060000 | 1870   |
      | sell | 1200070000 | 1874   |
      | sell | 1200080000 | 1877   |
      | sell | 1200090000 | 1881   |
      | sell | 1200100000 | 1884   |

  Scenario:  LP pegged volume is pushed inside price monitoring bounds(0034-PROB-002, 0009-MRKP-003, 0009-MRKP-004, 0009-MRKP-005);

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau     | mu | r | sigma |
      | 0.000001      | 0.00273 | 0  | 0 | 1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 43200   | 0.982       | 300               |
    And the markets:
      | id         | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1    | ETH2  | 100000000 |
      | party1 | ETH2  | 10000000  |
      | party2 | ETH2  | 10000000  |

    And the parties submit the following liquidity provision:
      | id          | party | market id  | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 100    | submission |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 90     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 80     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 70     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 60     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 50     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 40     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 30     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 20     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 40     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 50     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 60     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 70     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 80     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 90     | amendment  |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 100    | amendment  |

    And the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH2/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH2/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH2/MAR22 | sell | 1      | 1109  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH2/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH2/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 900       | 1109      | 3611         | 50000000       | 10            |

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume |
      | sell | 1109  | 90177  |
      | sell | 1099  | 0      |
      | sell | 1089  | 0      |
      | sell | 1079  | 0      |
      | sell | 1069  | 0      |
      | sell | 1059  | 0      |
      | sell | 1049  | 0      |
      | sell | 1049  | 0      |
      | sell | 1029  | 0      |
      | sell | 1019  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 0      |
      | buy  | 920   | 0      |
      | buy  | 900   | 111113 |
      | buy  | 920   | 0      |
      | buy  | 880   | 0      |

    # at this point what's left on the book is the buy @ 900 and sell @ 1109
    # so the best bid/ask coincides with the price monitoring bounds.
    # Since the lp1 offset is +/- 100 (depending on side) the lp1 volume "should" go to 800 and 1209
    # but because the price monitoring bounds are 900 and 1109 the volume gets pushed to these
    # i.e. it's placed at 900 / 1109.

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search   | initial  | release  |
      | lp1   | ETH2/MAR22 | 32570956    | 35828051 | 39085147 | 45599338 |

    And the parties should have the following account balances:
      | party | asset | market id  | margin   | general  | bond     |
      | lp1   | ETH2  | ETH2/MAR22 | 39085147 | 10914853 | 50000000 |

    Then the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH2/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3 |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 900       | 1109      | 3611         | 50000000       | 10            |
    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume |
      | sell | 1109  | 90177  |
      | buy  | 901   | 0      |
      | buy  | 900   | 111114 |

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search   | initial  | release  |
      | lp1   | ETH2/MAR22 | 32570956    | 35828051 | 39085147 | 45599338 |

    # now we place an order which makes the best bid 901.
    Then the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH2/MAR22 | buy  | 1      | 901   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4 |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 900       | 1109      | 3611         | 50000000       | 10            |

    # the lp1 one volume on this side should go to 801 but because price monitoring bound is still 900 it gets pushed to 900.
    # but 900 is no longer the best bid, so the risk model is used to get prob of trading. This now given by the log-normal model
    # Hence a bit volume is required to meet commitment and thus the margin requirement moves but not much.

    Then the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume |
      | sell | 1109  | 90177  |
      | buy  | 901   | 1      |
      | buy  | 900   | 112674 |
      | buy  | 899   | 0      |

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search   | initial  | release  |
      | lp1   | ETH2/MAR22 | 32570956    | 35828051 | 39085147 | 45599338 |

  Scenario:  LP pegged volume is pushed by Price Monitoring lower bound (0034-PROB-003);

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau     | mu | r | sigma |
      | 0.000001      | 0.00273 | 0  | 0 | 1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability    | auction extension |
      | 43200   | 0.999999999999 | 300               |
    And the markets:
      | id         | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | lp1    | ETH2  | 100000000000000 |
      | party1 | ETH2  | 10000000        |
      | party2 | ETH2  | 10000000        |

    And the parties submit the following liquidity provision:
      | id          | party | market id  | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 250    | submission |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 250    | amendment  |

    And the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH2/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH2/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH2/MAR22 | sell | 1      | 1109  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH2/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH2/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 728       | 1371      | 3611         | 50000000       | 10            |

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume    |
      | sell | 1359  | 67560669  |
      | sell | 1109  | 1         |
      | sell | 909   | 0         |
      | buy  | 1000  | 0         |
      | buy  | 728   | 175050882 |
      | buy  | 900   | 1         |

  Scenario:  LP Volume being pushed by limit of Probability of Trading (capped at 1e-8) (0034-PROB-004)
    #Price Monitoring has been removed as Prob in Price Monitoring only take up to 15 decimal places which will prevent scenatio which will trigger the ProbOfTrading cap at 1e-8

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau     | mu | r | sigma |
      | 0.000001      | 0.00273 | 0  | 0 | 1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |

    And the markets:
      | id         | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring | data source config     |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future |
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
      | side | price | volume         |
      | sell | 1709  | 2925687536572  |
      | sell | 1109  | 1              |
      | buy  | 300   | 16666666666667 |
      | buy  | 900   | 1              |

  # vol_LP_Short= 50000000/1709/1e-8=2.925e12
  # vol_LP_Long= 50000000/300/1e-8=1.66e13

  Scenario:  Create LP shape that pegs to mid and deploys volumes and price between best ask and best bid (0034-PROB-005)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |

    And the markets:
      | id         | quote name | asset | risk model                | margin calculator         | auction duration | fees          | price monitoring | data source config     |
      | ETH2/MAR22 | ETH2       | ETH2  | default-simple-risk-model | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future |
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
      | buy  | 95    | 221607 |
      #50000000*(20/95)/0.5/95
      | buy  | 90    | 350878 |
      #50000000*(30/95)/0.5/90
      | buy  | 85    | 495357 |
      #50000000*(40/95)/0.5/85
      | buy  | 80    | 65790  |
      #50000000*(5/95)/0.5/80
      | sell | 105   | 200502 |
      #50000000*(20/95)/0.5/105
      | sell | 110   | 287082 |
      #50000000*(30/95)/0.5/110
      | sell | 115   | 366133 |
      #50000000*(40/95)/0.5/115
      | sell | 120   | 43860  |
#50000000*(5/95)/0.5/120
