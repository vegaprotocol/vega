Feature: Spot market fees distribution

  Scenario: 0029-FEES-022: Fees are collected during continuous trading and auction modes and distributed to the appropriate accounts when Position Decimal Places (PDP) is -2

  Background:

    Given the following network parameters are set:
      | name                                            | value |
      | network.markPriceUpdateMaximumFrequency         | 1s    |
      | market.value.windowLength                       | 1h    |
      | market.liquidity.maximumLiquidityFeeFactorLevel | 0.4   |
      | validators.epoch.length                         | 4s    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36000   | 0.999       | 1                 |
      | 38000   | 0.999       | 2                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | -2                      | default-basic |

    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100000 |
      | party1 | BTC   | 100000 |
      | party2 | ETH   | 100000 |
      | party2 | BTC   | 100000 |
      | lp     | ETH   | 100000 |
      | lp     | BTC   | 100000 |
    And the average block duration is "1"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp    | BTC/ETH   | 500               | 0.025 | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 30    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | open interest |
      | 20         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 18        | 22        | 0             |
      | 20         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 38000   | 18        | 22        | 0             |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party2 | BTC/ETH   | sell | 1      | 20    | 1                | TYPE_LIMIT | TIF_GTC |             |
      | party1 | BTC/ETH   | buy  | 1      | 18    | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party2 | BTC/ETH   | sell | 1      | 22    | 0                | TYPE_LIMIT | TIF_GTC | sell-party2 |

    Then the network moves ahead "1" blocks
    #lp fee: 2000*100*0.025=50
    And the following transfers should happen:
      | from   | to     | from account                | to account                       | market id | amount | asset |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 20     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 60     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 50     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER     | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 20     | ETH   |
      | market | lp     | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | BTC/ETH   | 50     | ETH   |

    And the parties amend the following orders:
      | party  | reference   | price | size delta | tif     |
      | party2 | sell-party2 | 18    | 0          | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | last traded price | trading mode            |
      | 20         | 18                | TRADING_MODE_CONTINUOUS |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 18    | 1    | party2 |

    Then the network moves ahead "2" blocks
    And the following transfers should happen:
      | from   | to     | from account                | to account                       | market id | amount | asset |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 18     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 54     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 45     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER     | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 18     | ETH   |
      | market | lp     | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | BTC/ETH   | 45     | ETH   |

