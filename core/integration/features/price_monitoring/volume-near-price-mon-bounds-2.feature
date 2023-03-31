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
      | network.markPriceUpdateMaximumFrequency             | 0s    |

    And the average block duration is "1"

  Scenario: second scenario for volume at near price monitoring bounds with log-normal

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
      | id         | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future | 1e6                    | 1e6                       |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name              | value  |
      | prices.ETH2.value | 100000 |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | lp1    | ETH2  | 10000000000 |
      | party1 | ETH2  | 1000000000  |
      | party2 | ETH2  | 1000000000  |

    Given the parties submit the following liquidity provision:
      | id          | party | market id  | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | commitment1 | lp1   | ETH2/MAR22 | 3000000           | 0.001 | buy  | BID              | 500        | 100    | submission |
      | commitment1 | lp1   | ETH2/MAR22 | 3000000           | 0.001 | sell | ASK              | 500        | 100    | amendment  |

    And the parties place the following orders:
      | party  | market id  | side | volume | price  | resulting trades | type       | tif     | reference  |
      | party1 | ETH2/MAR22 | buy  | 1      | 89942  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH2/MAR22 | buy  | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH2/MAR22 | sell | 1      | 110965 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH2/MAR22 | sell | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH2/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "100000"

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000     | TRADING_MODE_CONTINUOUS | 43200   | 89942     | 110965    | 361190       | 3000000        | 10            |

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price  | volume |
      | sell | 111065 | 28     |
      | sell | 110965 | 1      |
      | buy  | 89943  | 0      |
      | buy  | 89942  | 1      |
      | buy  | 89842  | 34     |

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search  | initial | release |
      | lp1   | ETH2/MAR22 | 1011341     | 1112475 | 1213609 | 1415877 |

    And the parties should have the following account balances:
      | party | asset | market id  | margin  | general    | bond    |
      | lp1   | ETH2  | ETH2/MAR22 | 1213609 | 9995786391 | 3000000 |

    Then the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH2/MAR22 | buy  | 1      | 89942 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3 |

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price  | volume |
      | sell | 111065 | 28     |
      | sell | 110965 | 1      |
      | buy  | 89943  | 0      |
      | buy  | 89942  | 2      |
      | buy  | 89842  | 34     |

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search  | initial | release |
      | lp1   | ETH2/MAR22 | 1011341     | 1112475 | 1213609 | 1415877 |

    # # now we place an order which makes the best bid 89943.
    Then the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH2/MAR22 | buy  | 1      | 89943 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4 |

    And the market data for the market "ETH2/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100000     | TRADING_MODE_CONTINUOUS | 43200   | 89942     | 110965    | 361190       | 3000000        | 10            |

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price  | volume |
      | sell | 111065 | 28     |
      | sell | 110965 | 1      |
      | buy  | 89943  | 1      |
      | buy  | 89942  | 2      |
      | buy  | 89843  | 34     |

    And the parties should have the following margin levels:
      | party | market id  | maintenance | search  | initial | release |
      | lp1   | ETH2/MAR22 | 1011341     | 1112475 | 1213609 | 1415877 |

