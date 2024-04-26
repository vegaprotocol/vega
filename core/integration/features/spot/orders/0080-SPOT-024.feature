Feature: Spot market bug (from the incentive) replication

  Scenario: 0080-SPOT-024, 0080-SPOT-025 market orders should be rejected when traders do not have enough collateral to cover the trades in spot market
  Background:

    Given the following network parameters are set:
      | name                                            | value |
      | network.markPriceUpdateMaximumFrequency         | 1s    |
      | market.value.windowLength                       | 1h    |
      | market.liquidity.maximumLiquidityFeeFactorLevel | 0.4   |
      | validators.epoch.length                         | 4s    |

    Given the following assets are registered:
      | id  | decimal places |
      | USD | 0              |
      | BTC | 0              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36000   | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/USD | BTC/USD | BTC        | USD         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 0              | 0                       | default-basic |

    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | USD   | 50000000 |
      | party1 | BTC   | 100      |
      | party2 | USD   | 1000     |
      | party3 | USD   | 1000     |
      | party5 | USD   | 70000    |
      | party2 | BTC   | 100      |
      | party4 | BTC   | 1        |
      | lp     | USD   | 1000     |
      | lp     | BTC   | 100      |
    And the average block duration is "1"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp    | BTC/USD   | 5                 | 0.025 | submission |

    Then "party2" should have general account balance of "1000" for asset "USD"
    Then "party2" should have general account balance of "100" for asset "BTC"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/USD   | buy  | 10     | 55000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/USD   | buy  | 3      | 58000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/USD   | buy  | 3      | 59000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/USD   | buy  | 1      | 60000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 1      | 60000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 3      | 61000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 3      | 62000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 4      | 63000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 10     | 64000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/USD   | sell | 5      | 65000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/USD" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | open interest |
      | 60000      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 52466     | 68503     | 0             |

    Then "party2" should have general account balance of "61000" for asset "USD"
    Then "party2" should have general account balance of "74" for asset "BTC"

    Then "party2" should have holding account balance of "25" for asset "BTC"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference | error                        |
      | party3 | BTC/USD   | buy  | 20     | 0     | 0                | TYPE_MARKET | TIF_IOC | p3-b1     | insufficient funds for order |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference | error                                                              |
      | party4 | BTC/USD   | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_IOC | p4-s1     | party does not have sufficient balance to cover the trade and fees |

    Then the network moves ahead "1" blocks

    Then "party2" should have general account balance of "61000" for asset "USD"
    Then "party2" should have general account balance of "74" for asset "BTC"

    Then "party2" should have holding account balance of "25" for asset "BTC"

    Then "party3" should have general account balance of "1000" for asset "USD"
    Then "party4" should have general account balance of "1" for asset "BTC"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party3 | BTC/USD   | buy  | 20     | 10    | 0                | TYPE_LIMIT | TIF_GTC | p3-b2     |       |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party4 | BTC/USD   | sell | 1      | 66000 | 0                | TYPE_LIMIT | TIF_GTC | p4-s2     |       |

    #0080-SPOT-026: amend order - order is amended such that would trade immediately and the party can't afford none/some of the trades
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error                                                              |
      | party3 | p3-b2     | 66000 | -10        | TIF_GTC | party does not have sufficient balance to cover the trade and fees |
      | party4 | p4-s2     | 59000 | 2          | TIF_GTC | party does not have sufficient balance to cover the new size       |

    And the order book should have the following volumes for market "BTC/USD":
      | side | price | volume |
      | buy  | 10    | 20     |
      | sell | 66000 | 1      |

    #0080-SPOT-027:submit order - limit order, partly matched, party can't afford the trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                              |
      | party3 | BTC/USD   | buy  | 6      | 61000 | 0                | TYPE_LIMIT | TIF_GTC | p3-b3     | party does not have sufficient balance to cover the trade and fees |
      | party5 | BTC/USD   | buy  | 2      | 61000 | 0                | TYPE_LIMIT | TIF_GTC | p5-b1     | party does not have sufficient balance to cover the trade and fees |

    #0080-SPOT-028:submit order - limit order, no match, added to the book, party can't cover the amount that needs to be transfered to the holding
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                              |
      | party4 | BTC/USD   | sell | 2      | 67000 | 0                | TYPE_LIMIT | TIF_GTC | p4-s3     | party does not have sufficient balance to cover the trade and fees |

