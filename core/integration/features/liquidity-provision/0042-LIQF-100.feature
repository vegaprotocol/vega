Feature: Test LP mechanics when there are multiple liquidity providers, test stop-loss orders and parked orders (does not count towards LP commitment)

  Background:
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the following network parameters are set:
      | name                                    | value |
      | market.value.windowLength               | 60s   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 6     |
      | market.auction.minimumDuration          | 1     |
      | market.fee.factors.infrastructureFee    | 0.001 |
      | market.fee.factors.makerFee             | 0.004 |
      | spam.protection.max.stopOrdersPerMarket | 5     |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 20s         | 1              |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    ## Set auction duration to 3 epochs
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 30                |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |
    And the liquidity sla params named "SLA-23":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0           | 0.6                          | 1                             | 1.0                    |

    And the spot markets:
      | id      | name    | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lqm-params           | log-normal-risk-model | 2                | fees-config-1 | price-monitoring | SLA-22     |


    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.1   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.25  |

    Given the average block duration is "1"
  @Now
  Scenario: 001: lp1 and lp2 on the market BTC/ETH, 0044-LIME-078, 0042-LIQF-100
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | ETH   | 100000 |
      | lp2    | ETH   | 100000 |
      | lp3    | ETH   | 100000 |
      | party1 | ETH   | 100000 |
      | party2 | ETH   | 100000 |
      | party3 | ETH   | 100000 |
      | ptbuy  | ETH   | 100000 |
      | ptsell | ETH   | 100000 |
      | lp1    | BTC   | 100000 |
      | lp2    | BTC   | 100000 |
      | lp3    | BTC   | 100000 |
      | party1 | BTC   | 100000 |
      | party2 | BTC   | 100000 |
      | party3 | BTC   | 100000 |
      | ptbuy  | BTC   | 100000 |
      | ptsell | BTC   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | BTC/ETH   | 6000              | 0.02  | submission |
      | lp_2 | lp2   | BTC/ETH   | 4000              | 0.015 | submission |

    When the network moves ahead "4" blocks
    And the current epoch is "0"

    # AC: 0042-LIQF-054: If an LP has an active liquidity provision at the start of an epoch and no previous performance penalties and throughout the epoch always meets their liquidity provision requirements
    # then they will have a `fraction_of_time_on_book == 1` then no penalty will be applied to their liquidity fee payments at the end of the epoch.
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | BTC/ETH   | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | lp1    | BTC/ETH   | buy  | 10     | 950   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     |
      | lp2    | BTC/ETH   | buy  | 10     | 970   | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     |
      | lp2    | BTC/ETH   | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | lp2-s     |
      | lp1    | BTC/ETH   | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     |

    Then the opening auction period ends for market "BTC/ETH"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 10000        | 10000          |

    When the network moves ahead "1" epochs
    And the supplied stake should be "10000" for the market "BTC/ETH"
    And the network treasury balance should be "0" for the asset "ETH"
    And the current epoch is "1"
    Then the parties should have the following account balances:
      | party | asset | market id | general | bond |
      | lp1   | ETH   | BTC/ETH   | 84500   | 6000 |
      | lp2   | ETH   | BTC/ETH   | 86300   | 4000 |

    Then the parties cancel the following orders:
      | party | reference |
      | lp1   | lp1-b     |
      | lp2   | lp2-b     |
      | lp2   | lp2-s     |
      | lp1   | lp1-s     |
    Then the network moves ahead "1" blocks

    #AC 0044-LIME-077: Parked pegged limit orders and stop-loss orders do not count towards an LPs liquidity commitment.
    # post-only orders count towards an LPs liquidity commitment
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference | only |
      | lp1   | BTC/ETH   | buy  | 10     | 950   | 0                | TYPE_LIMIT | TIF_GTC | lp1-b     | post |
      | lp2   | BTC/ETH   | buy  | 10     | 970   | 0                | TYPE_LIMIT | TIF_GTC | lp2-b     | post |
      | lp2   | BTC/ETH   | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | lp2-s     | post |
      | lp1   | BTC/ETH   | sell | 10     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | lp1-s     | post |

    When the network moves ahead "1" epochs
    And the supplied stake should be "10000" for the market "BTC/ETH"

    Then the parties cancel the following orders:
      | party | reference |
      | lp1   | lp1-b     |
      | lp1   | lp1-s     |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1   | BTC/ETH   | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | BTC/ETH   | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" blocks

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | reference        | only   | fb price trigger |
      | lp1   | BTC/ETH   | buy  | 2      | 950   | 0                | TYPE_MARKET | TIF_GTC | lp1-b-stop-order | reduce | 900              |

    When the network moves ahead "1" epochs

    Then the supplied stake should be "9400" for the market "BTC/ETH"
    And the current epoch is "3"
    And the network treasury balance should be "600" for the asset "ETH"
    And the following transfers should happen:
      | from | to     | from account      | to account                    | market id | amount | asset |
      | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 600    | ETH   |

    ## Trigger price monitoring auction by trading outside of price bound (973-1027)
    ## Ensure volume on the book after leaving auction at 900-990
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | ptbuy  | BTC/ETH   | buy  | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | BTC/ETH   | sell | 2      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptbuy  | BTC/ETH   | sell | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptsell | BTC/ETH   | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 10000        | 9400           | 30          |

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | BTC/ETH   | 12        | 1                    | buy  | BID              | 12     | 20     | lp1-b     |
      | lp1   | BTC/ETH   | 12        | 1                    | sell | ASK              | 12     | 20     | lp1-s     |
    When the network moves ahead "1" blocks

    And the orders should have the following status:
      | party | reference | status        |
      | lp1   | lp1-b     | STATUS_PARKED |
      | lp1   | lp1-s     | STATUS_PARKED |

    When the network moves ahead "1" epochs
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | target stake | supplied stake | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 10000        | 8860           | 30          |

    #lp1 got bond penalty for placing parked order
    Then the following transfers should happen:
      | from | to     | from account      | to account                    | market id | amount | asset |
      | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 540    | ETH   |

    And the network treasury balance should be "1140" for the asset "ETH"


