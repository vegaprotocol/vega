# Test volume and margin when LP volume is pushed inside price monitoring bounds
# and the price monitoring bounds happen to be best bid/ask
Feature: Test margin for lp near price monitoring boundaries

  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |

    And the average block duration is "1"

  Scenario: first scenario for volume at near price monitoring bounds and simple-risk-model

    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2021-12-31T23:59:59Z |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1     | ETH   | 100000000 |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |

    Given the parties submit the following liquidity provision:
      | id          | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | commitment1 | lp1   | ETH/DEC21 | 78000000          | 0.001 | buy  | BID              | 500        | 100    | submission |
      | commitment1 | lp1   | ETH/DEC21 | 78000000          | 0.001 | sell | ASK              | 500        | 100    | amendment |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 900       | 1100      | 1000         | 78000000       | 10            |


    # at this point what's left on the book is the buy @ 900 and sell @ 1100
    # so the best bid/ask coincides with the price monitoring bounds.
    # Since the lp1 offset is +/- 100 (depending on side) the lp1 volume "should" go to 800 and 1200
    # but because the price monitoring bounds are 900 and 1100 the volume gets pushed to these
    # i.e. it's placed at 900 / 1100.
    # As these are the best bid / best ask the probability of trading used is 1/2.

    And the parties should have the following margin levels:
      | party | market id | maintenance | search   | initial  | release  |
      | lp1    | ETH/DEC21 | 17333400    | 19066740 | 20800080 | 24266760 |

    And the parties should have the following account balances:
      | party | asset | market id | margin   | general | bond     |
      | lp1    | ETH   | ETH/DEC21 | 20800080 | 1199920 | 78000000 |


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3 |

    And the parties should have the following margin levels:
      | party | market id | maintenance | search   | initial  | release  |
      | lp1    | ETH/DEC21 | 17333400    | 19066740 | 20800080 | 24266760 |

    # now we place an order which makes the best bid 901.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | buy  | 1      | 901   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4 |

    # the lp1 one volume on this side should go to 801 but because price monitoring bound is still 900 it gets pushed to 900.
    # but 900 is no longer the best bid, so the risk model is used to get prob of trading. This is 0.1 (see above).
    # Hence a lot more volume is required to meet commitment and thus the margin requirement jumps substantially.

    And the parties should have the following margin levels:
      | party | market id | maintenance | search   | initial   | release   |
      | lp1    | ETH/DEC21 | 86666700    | 95333370 | 104000040 | 121333380 |

  Scenario: second scenario for volume at near price monitoring bounds with log-normal

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau     | mu | r | sigma |
      | 0.000001      | 0.00273 | 0  | 0 | 1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 43200   | 0.982       | 300               |
    And the markets:
      | id         | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future | 2022-03-31T23:59:59Z |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1     | ETH2  | 100000000 |
      | party1 | ETH2  | 10000000  |
      | party2 | ETH2  | 10000000  |

    And the parties submit the following liquidity provision:
      | id          | party | market id  | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | buy  | BID              | 500        | 100    | submission |
      | commitment1 | lp1   | ETH2/MAR22 | 50000000          | 0.001 | sell | ASK              | 500        | 100    | amendment |
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
      | sell | 1109  | 90173  |
      | buy  | 901   | 0      |
      | buy  | 900   | 111113 |


    # at this point what's left on the book is the buy @ 900 and sell @ 1109
    # so the best bid/ask coincides with the price monitoring bounds.
    # Since the lp1 offset is +/- 100 (depending on side) the lp1 volume "should" go to 800 and 1209
    # but because the price monitoring bounds are 900 and 1109 the volume gets pushed to these
    # i.e. it's placed at 900 / 1109.
    # As these are the best bid / best ask the probability of trading used is 1/2.

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search   | initial  | release  |
      | lp1    | ETH2/MAR22 | 32569511    | 35826462 | 39083413 | 45597315 |

    And the parties should have the following account balances:
      | party | asset | market id  | margin   | general  | bond     |
      | lp1    | ETH2  | ETH2/MAR22 | 39083413 | 10916587 | 50000000 |

    Then the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH2/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3 |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 900       | 1109      | 3611         | 50000000       | 10            |
    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price | volume |
      | sell | 1109  | 90173  |
      | buy  | 901   | 0      |
      | buy  | 900   | 111114 |


    And the parties should have the following margin levels:
      | party | market id  | maintenance | search   | initial  | release  |
      | lp1    | ETH2/MAR22 | 32569511    | 35826462 | 39083413 | 45597315 |

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
      | side | price    | volume |
      | sell | 1109     | 90173  |
      | buy  | 901      | 1      |
      | buy  | 900      | 112668 |
      | buy  | 899      | 0      |

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search   | initial  | release  |
      | lp1   | ETH2/MAR22 | 32569511    | 35826462 | 39083413 | 45597315 |
