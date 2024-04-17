Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_CONSTANT      | 0.02                   |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 1.0              | 20s         | 1              |

    And the following network parameters are set:
      | name                                          | value |
      | market.value.windowLength                     | 60s   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | limits.markets.maxPeggedOrders                | 6     |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.5                          | 1                             | 1.0                    |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model        | auction duration | fees          | price monitoring | sla params | liquidity monitoring |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | simple-risk-model | 2                | fees-config-1 | price-monitoring | SLA        | lqm-params           |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.5   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |

    Given the average block duration is "2"
  @Now
  Scenario: 001: lp1 and lp2 under supplies liquidity
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | ETH   | 100000 |
      | lp2    | ETH   | 100000 |
      | party1 | ETH   | 100000 |
      | lp1    | BTC   | 100    |
      | lp2    | BTC   | 100    |
      | party2 | BTC   | 100    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type    |
      | lp_1 | lp1   | BTC/ETH | 50000             | 0.02  | submission |
      | lp_2 | lp2   | BTC/ETH | 10000             | 0.015 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | BTC/ETH | 12        | 1                    | buy  | BID              | 12     | 20     | lp-b-1    |
      | lp1   | BTC/ETH | 12        | 1                    | sell | ASK              | 12     | 20     | lp-s-1    |
      | lp2   | BTC/ETH | 6         | 1                    | buy  | BID              | 6      | 20     | lp-b-2    |
      | lp2   | BTC/ETH | 6         | 1                    | sell | ASK              | 6      | 20     | lp-s-2    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH | buy  | 1     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 1     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"
    And the supplied stake should be "60000" for the market "BTC/ETH"

    And the liquidity fee factor should be "0.02" for the market "BTC/ETH"

    And the parties should have the following account balances:
      | party | asset | market id | general | bond  |
      | lp1   | ETH   | BTC/ETH   | 50000   | 50000 |
      | lp2   | ETH   | BTC/ETH   | 37200   | 10000 |
    Then the network moves ahead "6" blocks
    And the parties should have the following account balances:
      | party | asset | market id | general | bond  |
      | lp1   | ETH   | BTC/ETH   | 50000   | 25000 |
      | lp2   | ETH   | BTC/ETH   | 37200   | 10000 |

  @SLABug
  Scenario: 002: lp1 and lp2 amend LP commitment
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | ETH   | 100000  |
      | lp2    | ETH   | 100000  |
      | party1 | ETH   | 100000  |
      | lp1    | BTC   | 100     |
      | lp2    | BTC   | 100     |
      | party2 | BTC   | 100     |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | BTC/ETH | 50000             | 0.02 | submission |
      | lp_2 | lp2   | BTC/ETH | 10000             | 0.01 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | BTC/ETH | 120       | 1                    | buy  | BID              | 120    | 20     | lp-b-1    |
      | lp1   | BTC/ETH | 120       | 1                    | sell | ASK              | 120    | 20     | lp-s-1    |
      | lp2   | BTC/ETH | 60        | 1                    | buy  | BID              | 60     | 20     | lp-b-2    |
      | lp2   | BTC/ETH | 60        | 1                    | sell | ASK              | 60     | 20     | lp-s-2    |
    Then the network moves ahead "2" blocks
    And the orders should have the following status:
      | party | reference | status        |
      | lp1   | lp-b-1    | STATUS_PARKED |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH | buy  | 1     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 1     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      #Then the network moves ahead "2" blocks

    Then the opening auction period ends for market "BTC/ETH"

    And the orders should have the following status:
      | party | reference | status        |
      | lp1   | lp-b-1    | STATUS_ACTIVE |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the parties should have the following account balances:
      | party | asset | market id | general | bond  |
      | lp1   | ETH   | BTC/ETH   | 309757  | 50000 |
      | lp2   | ETH   | BTC/ETH   | 309757  | 50000 |

    #AC: 0044-LIME-105, lp reduces commitment
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | BTC/ETH | 30000             | 0.02 | amendment |
    And the supplied stake should be "60000" for the market "BTC/ETH"
    Then the network moves ahead "10" blocks
    And the supplied stake should be "30000" for the market "BTC/ETH"

    #AC: 0044-LIME-106, lp reduces commitment multi times
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | BTC/ETH | 28000             | 0.02 | amendment |
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | BTC/ETH | 27000             | 0.02 | amendment |
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | BTC/ETH | 27000             | 0.02 | amendment |
    And the supplied stake should be "40000" for the market "BTC/ETH"
    Then the network moves ahead "10" blocks
    And the supplied stake should be "37000" for the market "BTC/ETH"
    #AC: 0044-LIME-112, lp reduces commitment, no penalty
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | BTC/ETH | 640243 | 332757  | 27000 |

    #AC:0044-LIME-108, lp changes fee factor
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee   | lp type   |
      | lp_1 | lp1   | BTC/ETH | 50000             | 0.008 | amendment |
    And the liquidity fee factor should be "0.01" for the market "BTC/ETH"
    Then the network moves ahead "10" blocks
    And the liquidity fee factor should be "0.008" for the market "BTC/ETH"

    #AC: 0044-LIME-109, lp increases commitment and they do not have sufficient collateral in the settlement asset
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   | error                             |
      | lp_1 | lp1   | BTC/ETH | 600000            | 0.02 | amendment | commitment submission rejected, not enough stake |
    Then the network moves ahead "1" blocks
    And the supplied stake should be "60000" for the market "BTC/ETH"

    #AC: 0044-LIME-110, lp increases commitment and they have sufficient collateral in the settlement asset
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | BTC/ETH | 60000             | 0.02 | amendment |
    And the supplied stake should be "70000" for the market "BTC/ETH"

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | BTC/ETH | 640243 | 299757  | 60000 |
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3556         | 70000          | 1             |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 5      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | buy  | 1      | 960   | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | BTC/ETH | sell | 1      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
    And the liquidity fee factor should be "0.008" for the market "BTC/ETH"
    #liquidity fee collected: 5*1000*0.008=40

    #AC: 0044-LIME-107, lp decreases commitment and gets bond slashing
    #AC: 0044-LIME-111, at the end of the current epoch rewards/penalties are evaluated based on the balance of the bond account at start of epoch
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | BTC/ETH | 640243 | 299757  | 60000 |
      | lp2   | USD   | BTC/ETH | 320122 | 669878  | 10000 |
    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type   |
      | lp_1 | lp1   | BTC/ETH | 1000              | 0.02 | amendment |
      | lp_2 | lp2   | BTC/ETH | 500               | 0.02 | amendment |
    Then the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | BTC/ETH | 640243 | 309757  | 50000 |
      | lp2   | USD   | BTC/ETH | 320122 | 669878  | 10000 |
    # trigger price auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | ptsell | BTC/ETH | sell | 1      | 940   | 0                | TYPE_LIMIT | TIF_GTC |
      | ptbuy  | BTC/ETH | buy  | 1      | 940   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | BTC/ETH | 0      | 950000  | 50000 |
      | lp2   | USD   | BTC/ETH | 0      | 990000  | 10000 |
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 20274        | 60000          | 6             | 3           |
    When the network moves ahead "10" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | BTC/ETH | 608232 | 386874  | 1002 |
      | lp2   | USD   | BTC/ETH | 304116 | 694629  | 501  |
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest | horizon | min bound | max bound |
      | 950        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 20274        | 1503           | 6             | 3600    | 925       | 976       |

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH | 33     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH | 6      | USD   |
