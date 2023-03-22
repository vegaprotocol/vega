Feature: Test for issue 5460

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 1     |
      | network.floatingPointUpdates.delay            | 10s   |
      | market.auction.minimumDuration                | 1     |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.001     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 43200   | 0.9999999   | 60                |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 10               | fees-config-1 | price-monitoring-1 | default-eth-for-future | 5              | 5                       | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount            |
      | party0   | ETH   | 100000000000000   |
      | party1   | ETH   | 10000000000000    |
      | party2   | ETH   | 10000000000000    |
      | party3   | ETH   | 10000000000000    |
      | party_a1 | ETH   | 10000000000000    |
      | party_a2 | ETH   | 10000000000000    |
      | party_r  | ETH   | 10000000000000000 |
      | party_r1 | ETH   | 10000000000000000 |

  Scenario: 002 replicate bug

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 200000000         | 0.001 | buy  | MID              | 2          | 205    | submission |
      | lp1 | party0 | ETH/DEC21 | 200000000         | 0.001 | sell | MID              | 2          | 205    | submission |

    And the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | party_a1 | ETH/DEC21 | buy  | 100000 | 30000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party_a2 | ETH/DEC21 | sell | 100000 | 30000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party_r  | ETH/DEC21 | buy  | 100000 | 29998 | 0                | TYPE_LIMIT | TIF_GTC |
      | party_r  | ETH/DEC21 | sell | 100000 | 30002 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the auction ends with a traded volume of "100000" at a price of "30000"

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 30000      | TRADING_MODE_CONTINUOUS | 43200   | 24617     | 36510     | 1626         | 200000000      | 100000        |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party_r | ETH/DEC21 | buy  | 100000 | 29987 | 0                | TYPE_LIMIT | TIF_GTC |
      | party_r | ETH/DEC21 | buy  | 100000 | 29977 | 0                | TYPE_LIMIT | TIF_GTC |
      | party_r | ETH/DEC21 | buy  | 100000 | 29967 | 0                | TYPE_LIMIT | TIF_GTC |
      | party_r | ETH/DEC21 | buy  | 100000 | 29957 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 30000      | TRADING_MODE_CONTINUOUS | 43200   | 24617     | 36510     | 1626         | 200000000      | 100000        |

    And the parties place the following orders:
      | party    | market id | side | volume | price  | resulting trades | type        | tif     |
      | party_r1 | ETH/DEC21 | buy  | 300000 | 400000 | 2                | TYPE_MARKET | TIF_IOC |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 29998 | 100000 |
      | buy  | 29987 | 100000 |
      | buy  | 29977 | 100000 |
      | buy  | 29967 | 100000 |
      | buy  | 29957 | 100000 |
      | buy  | 29795 | 0      |
      | sell | 30002 | 0      |
      | sell | 30205 | 0      |
    When the network moves ahead "1" blocks

    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC21"

    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party_r | ETH/DEC21 | sell | 100000 | 30002 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks

    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume    |
      | buy  | 29998 | 100000    |
      | buy  | 29987 | 100000    |
      | buy  | 29977 | 100000    |
      | buy  | 29967 | 100000    |
      | buy  | 29957 | 100000    |
      | buy  | 29795 | 671253567 |
      | sell | 30002 | 100000    |
      | sell | 30205 | 662142030 |

    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party_r | ETH/DEC21 | buy  | 100000 | 29700 | 0                | TYPE_LIMIT | TIF_GTC |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price  | volume    |
      | buy  | 29998  | 100000    |
      | buy  | 29987  | 100000    |
      | buy  | 29977  | 100000    |
      | buy  | 29967  | 100000    |
      | buy  | 29957  | 100000    |
      | buy  | 29795  | 671253567 |
      | buy  | 400000 | 0         |
      | buy  | 29700  | 100000    |
      | sell | 30002  | 100000    |
      | sell | 30205  | 662142030 |

    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 6549         | 200000000      | 400000        |

    And the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type        | tif     |
      | party_r1 | ETH/DEC21 | sell | 600000 | 29000 | 6                | TYPE_MARKET | TIF_IOC |

    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 8187         | 200000000      | 500000        |
