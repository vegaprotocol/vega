Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 40                |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.05        | 1                            | 1                             | 1.0                    |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 10               | 20s         | 0.1            |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 60s   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 6     |
      | market.auction.minimumDuration                      | 1     |
      | market.fee.factors.infrastructureFee                | 0.001 |
      | market.fee.factors.makerFee                         | 0.004 |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.5   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.providersFeeCalculationTimeStep    | 10s   |
    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.5              | 20s         | 1.0            |

    And the spot markets:
      | id      | name    | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring | sla params    | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | default-basic | SLA        |

    Given the average block duration is "2"

  Scenario: An LP with orders inside valid range during auction isn't penalised (0044-LIME-116)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 20000000   |
      | party1 | ETH   | 1000000000 |
      | party2 | ETH   | 1000000000 |
      | party3 | ETH   | 1000000    |
      | lp1    | BTC   | 20000000   |
      | party1 | BTC   | 1000000000 |
      | party2 | BTC   | 1000000000 |
      | party3 | BTC   | 1000000    |


    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | BTC/ETH   | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | BTC/ETH   | buy  | 100    | 4750  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | BTC/ETH   | sell | 100    | 5250  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 180000       | 180000         |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 180000       | 180000         | 40          |

    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | target stake | supplied stake |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | 180000       | 180000         |
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond  |
      | lp1   | ETH   | BTC/ETH   | 19340012 | 90000 |

    When the network moves ahead "1" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond  |
      | lp1   | ETH   | BTC/ETH   | 19340012 | 45000 |

  Scenario: An LP with bid orders outside valid range during auction is penalised (0044-LIME-117)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 20000000   |
      | party1 | ETH   | 1000000000 |
      | party2 | ETH   | 1000000000 |
      | party3 | ETH   | 1000000    |
      | lp1    | BTC   | 20000000   |
      | party1 | BTC   | 1000000000 |
      | party2 | BTC   | 1000000000 |
      | party3 | BTC   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | BTC/ETH   | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | BTC/ETH   | buy  | 100    | 4740  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | BTC/ETH   | sell | 100    | 5250  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 180000       | 180000         |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 180000       | 180000         | 40          |

    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond  |
      | lp1   | ETH   | BTC/ETH   | 19341023 | 90000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond  |
      | lp1   | ETH   | BTC/ETH   | 19341023 | 22500 |

  Scenario: An LP with ask orders outside valid range during auction is penalised (0044-LIME-118)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 20000000   |
      | party1 | ETH   | 1000000000 |
      | party2 | ETH   | 1000000000 |
      | party3 | ETH   | 1000000    |
      | lp1    | BTC   | 20000000   |
      | party1 | BTC   | 1000000000 |
      | party2 | BTC   | 1000000000 |
      | party3 | BTC   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | BTC/ETH   | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | BTC/ETH   | buy  | 100    | 4750  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | BTC/ETH   | sell | 100    | 5260  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 180000       | 180000         |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 180000       | 180000         | 40          |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond  |
      | lp1   | ETH   | BTC/ETH   | 19340012 | 90000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond  |
      | lp1   | ETH   | BTC/ETH   | 19340012 | 22500 |
