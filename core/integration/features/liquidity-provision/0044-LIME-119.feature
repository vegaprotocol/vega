Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | USD | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    # price monitoring duration should be > 3 epochs
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 40                |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.05        | 1                            | 1                             | 1.0                    |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 10               | 3600s       | 1              |

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
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.5              | 20s         | 1              |

    And the spot markets:
      | id      | name    | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring | sla params    | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | default-basic | SLA        |

    Given the average block duration is "2"

  @Now
  Scenario: An LP with bid orders inside valid range during auction (and market has no indicative price), is not penalised (0044-LIME-119)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 20000000   |
      | party1 | ETH   | 1000000000 |
      | party2 | ETH   | 1000000000 |
      | party3 | ETH   | 1000000    |
      | ptbuy  | ETH   | 10000000   |
      | ptsell | ETH   | 10000000   |
      | lp1    | BTC   | 20000000   |
      | party1 | BTC   | 1000000000 |
      | party2 | BTC   | 1000000000 |
      | party3 | BTC   | 1000000    |
      | ptbuy  | BTC   | 10000000   |
      | ptsell | BTC   | 10000000   |

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
    # Remove this so we can trigger price auction, we add it back later
    #| party1 | BTC/ETH | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 180000       | 180000         |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 5000   | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 5000   | 5000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 180000       | 180000         |

    ## Now trigger price monitoring auction - sell outside of max bound, and make sure orders can't uncross with existing volume
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | ptbuy  | BTC/ETH   | buy  | 1      | 4740  | 0                | TYPE_LIMIT | TIF_GTC | pt-buy    |
      | ptsell | BTC/ETH   | sell | 1      | 4740  | 0                | TYPE_LIMIT | TIF_GTC | pt-sell   |
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 180000       | 180000         | 40          |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond   |
      | lp1   | ETH   | BTC/ETH   | 19340012 | 162000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
    Then the parties should have the following account balances:
      | party | asset | market id | general  | bond   |
      | lp1   | ETH   | BTC/ETH   | 19340012 | 162000 |

Scenario: An LP with ask orders outside valid range during auction is penalised (0044-LIME-114)
  Given the parties deposit on asset's general account the following amount:
    | party  | asset | amount     |
    | lp1    | ETH   | 20000000   |
    | party1 | ETH   | 1000000000 |
    | party2 | ETH   | 1000000000 |
    | party3 | ETH   | 1000000    |
    | ptbuy  | ETH   | 10000000   |
    | ptsell | ETH   | 10000000   |
    | lp1    | BTC   | 20000000   |
    | party1 | BTC   | 1000000000 |
    | party2 | BTC   | 1000000000 |
    | party3 | BTC   | 1000000    |
    | ptbuy  | BTC   | 10000000   |
    | ptsell | BTC   | 10000000   |

  And the parties submit the following liquidity provision:
    | id   | party | market id | commitment amount | fee  | lp type    |
    | lp_1 | lp1   | BTC/ETH   | 180000            | 0.02 | submission |

  When the network moves ahead "2" blocks
  Then the parties place the following orders:
    | party  | market id | side | volume | price | resulting trades | type       | tif     |
    | lp1    | BTC/ETH   | buy  | 100    | 3790  | 0                | TYPE_LIMIT | TIF_GTC |
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
    | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
    | ptbuy  | BTC/ETH   | buy  | 1      | 4740  | 0                | TYPE_LIMIT | TIF_GTC | pt-buy    |
    | ptsell | BTC/ETH   | sell | 1      | 4740  | 0                | TYPE_LIMIT | TIF_GTC | pt-sell   |
  And the network moves ahead "2" blocks
  Then the market data for the market "BTC/ETH" should be:
    | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
    | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 180000       | 180000         | 40          |

  When the parties place the following orders:
    | party  | market id | side | volume | price | resulting trades | type       | tif     |
    | party1 | BTC/ETH   | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |
    | party1 | BTC/ETH   | buy  | 1      | 4000  | 0                | TYPE_LIMIT | TIF_GTC |
    | party2 | BTC/ETH   | sell | 1      | 4000  | 0                | TYPE_LIMIT | TIF_GTC |
  Then the network moves ahead "1" epochs
  And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
  And the parties should have the following account balances:
    | party | asset | market id | general  | bond  |
    | lp1   | ETH   | BTC/ETH   | 19437020 | 90000 |

  When the network moves ahead "2" epochs
  Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"
  Then the parties should have the following account balances:
    | party | asset | market id | general  | bond  |
    | lp1   | ETH   | BTC/ETH   | 19437020 | 22500 |

  Then the following transfers should happen:
    | from | to     | from account      | to account                    | market id | amount | asset |
    | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 90000  | ETH   |
    | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 45000  | ETH   |
    | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 22500  | ETH   |


