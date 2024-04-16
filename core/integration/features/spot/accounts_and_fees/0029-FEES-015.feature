Feature: Spot market fees distribution

  Scenario: 0029-FEES-015: Fees are collected during continuous trading and auction modes and distributed to the appropriate accounts, as described above.

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
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

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
      | party1 | BTC   | 10000  |
      | party2 | ETH   | 100000 |
      | party2 | BTC   | 10000  |
      | lp     | ETH   | 100000 |
      | lp     | BTC   | 10000  |
    And the average block duration is "1"

    # No orders have been places so we shouldn't have any holding accounts
    And "party1" should have only the following accounts:
      | type                 | asset | amount |
      | ACCOUNT_TYPE_GENERAL | ETH   | 100000 |
      | ACCOUNT_TYPE_GENERAL | BTC   | 10000  |

    And "party2" should have only the following accounts:
      | type                 | asset | amount |
      | ACCOUNT_TYPE_GENERAL | ETH   | 100000 |
      | ACCOUNT_TYPE_GENERAL | BTC   | 10000  |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp    | BTC/ETH   | 500               | 0.025 | submission |
    # Place some orders to create the holding accounts
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 100    | 3000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | open interest |
      | 2000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 1749      | 2283      | 0             |
      | 2000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 38000   | 1743      | 2291      | 0             |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party2 | BTC/ETH   | sell | 100    | 2000  | 1                | TYPE_LIMIT | TIF_GTC |             |
      | party1 | BTC/ETH   | buy  | 100    | 1800  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party2 | BTC/ETH   | sell | 100    | 2200  | 0                | TYPE_LIMIT | TIF_GTC | sell-party2 |

    Then the network moves ahead "1" blocks
    #lp fee: 2000*100*0.025=50
    And the following transfers should happen:
      | from   | to     | from account                | to account                       | market id | amount | asset |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 20     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 60     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 50     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER     | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 20     | ETH   |
      | market | lp     | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | BTC/ETH   | 50     | ETH   |

    #0029-FEES-016:Fees are collected in one case of amends: you amend the price so far that it causes an immediate trade.
    And the parties amend the following orders:
      | party  | reference   | price | size delta | tif     |
      | party2 | sell-party2 | 1800  | 0          | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | last traded price | trading mode            |
      | 2000       | 1800              | TRADING_MODE_CONTINUOUS |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1800  | 100  | party2 |

    Then the network moves ahead "2" blocks
    And the following transfers should happen:
      | from   | to     | from account                | to account                       | market id | amount | asset |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 18     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 54     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 45     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER     | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 18     | ETH   |
      | market | lp     | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | BTC/ETH   | 45     | ETH   |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 100    | 1745  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | BTC/ETH   | sell | 100    | 1745  | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 1800       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |
    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 1745       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    #The LP fees and infrastructure fees are split between the two parties for the trade during monitoring auction
    #lp fee: 1745*1*0.025 = 44
    And the following transfers should happen:
      | from   | to     | from account                | to account                       | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 27     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 27     | ETH   |
      | party1 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 22     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 22     | ETH   |
      | market | lp     | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | BTC/ETH   | 44     | ETH   |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | party1 | BTC/ETH   | buy  | 100    | 1780  | 0                | TYPE_LIMIT  | TIF_GTC |           |
      | party1 | BTC/ETH   | buy  | 23     | 1780  | 0                | TYPE_LIMIT  | TIF_GTC |           |
      | party2 | BTC/ETH   | sell | 123    | 1780  | 2                | TYPE_MARKET | TIF_IOC |           |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1780  | 100  | party2 |
      | party1 | 1780  | 23   | party2 |

    Then the network moves ahead "2" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 1780       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    #0029-FEES-021: market order fee distribution in spot market
    #liquidity fee: 1780*1.23*0.025=55
    And the following transfers should happen:
      | from   | to     | from account                | to account                       | market id | amount | asset |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 5      | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 13     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 11     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 45     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER     | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 5      | ETH   |
      | market | lp     | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES   | BTC/ETH   | 56     | ETH   |




