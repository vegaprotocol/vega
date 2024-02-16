Feature: Test LP mechanics when there are multiple liquidity providers;

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
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
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 40                |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.00001     | 0.5                          | 1                             | 1.0                    |

    And the liquidity sla params named "SLA2":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.5                          | 1                             | 1.0                    |

    And the liquidity sla params named "SLA3":
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
      | market.liquidity.equityLikeShareFeeFraction         | 1     |

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.5              | 20s         | 1.0            |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params | liquidity monitoring |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA        | lqm-params           |
      | ETH/MAR23 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA2       | lqm-params           |
      | ETH/JAN23 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA3       | lqm-params           |

    Given the average block duration is "2"
  @Now
  Scenario: 001: lp1 and lp2 under supplies liquidity (and expects to get penalty for not meeting the SLA) since both have orders outside price range
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 200000 |
      | lp2    | USD   | 15000  |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 80000             | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 500               | 0.01 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 2      | 200    | lp-b-1    |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 2      | 200    | lp-s-1    |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 2      | 200    | lp-b-2    |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 2      | 200    | lp-s-2    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 80500          | 1             |
    # # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569036

    And the liquidity fee factor should be "0.02" for the market "ETH/MAR22"

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 10671  | 109329  | 80000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 3829    | 500   |
    #margin_intial lp2: 2*1000*3.5569036*1.5=10671
    #lp1: 21342+98658+80000=200000; lp2: 10671+3829+500=15000

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 7113         | 80500          | 2             |

    Then the network moves ahead "6" blocks

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 10671  | 109329  | 40000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 3829    | 250   |
    #liquidity fee: 1000*0.02 = 20; lp1 get 19, lp2 get 0

    Then the following transfers should happen:
      | from   | to     | from account                   | to account                     | market id | amount | asset |
      | market | lp1    | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 19     | USD   |
      | lp1    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/MAR22 | 40000  | USD   |
      | lp2    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/MAR22 | 250    | USD   |
      | lp1    | market | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_INSURANCE         | ETH/MAR22 | 19     | USD   |

    And the insurance pool balance should be "40269" for the market "ETH/MAR22"

    Then the network moves ahead "6" blocks

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 10671  | 109329  | 20000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 3829    | 125   |

    And the insurance pool balance should be "60394" for the market "ETH/MAR22"
  # #increament in insurancepool: 60394-40269=20125 which is coming from SLA penalty on lp1 and lp2

  Scenario: An LP cancelled in one Epoch will receive no fees in the next even with orders still on-book (0044-LIME-097)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | lp1    | USD   | 2000000 |
      | lp2    | USD   | 2000000 |
      | party1 | USD   | 100000  |
      | party2 | USD   | 100000  |
      | party3 | USD   | 100000  |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR23 | 50000             | 0.02 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR23 | 2         | 1                    | buy  | BID              | 100    | 0      | lp-b-1    |
      | lp1   | ETH/MAR23 | 2         | 1                    | sell | ASK              | 100    | 0      | lp-s-1    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR23 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR23 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR23"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 50000          | 1             |

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR23 | 533536 | 1416464 | 50000 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" epochs

    # LP is paid
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR23 | 533536 | 1416484 | 50000 |

    Then the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_2 | lp2   | ETH/MAR23 | 50000             | 0.02 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp2   | ETH/MAR23 | 2         | 1                    | buy  | BID              | 100    | 0      | lp-b-2    |
      | lp2   | ETH/MAR23 | 2         | 1                    | sell | ASK              | 100    | 0      | lp-s-2    |

    And party "lp1" cancels their liquidity provision for market "ETH/MAR23"

    Then the network moves ahead "1" epochs
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR23 | 533536 | 1466484 | 0     |
      | lp2   | USD   | ETH/MAR23 | 533536 | 1416464 | 50000 |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR23 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" epochs
    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR23 | 533536 | 1466484 | 0     |
      | lp2   | USD   | ETH/MAR23 | 533536 | 1416484 | 50000 |

  Scenario: An LP with orders inside valid range during auction isn't penalised (0044-LIME-096)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 20000000   |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |
      | party3 | USD   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/JAN23 | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | ETH/JAN23 | buy  | 100    | 4750  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/JAN23 | sell | 100    | 5250  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/JAN23"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 17784        | 180000         | 1             |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 34679        | 180000         | 1             | 40          |

    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | 34679        | 180000         | 1             |
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17018938 | 90000 |

    When the network moves ahead "1" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17018938 | 45000 |

  Scenario: An LP with bid orders outside valid range during auction is penalised (0044-LIME-098)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 20000000   |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |
      | party3 | USD   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/JAN23 | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | ETH/JAN23 | buy  | 100    | 4740  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/JAN23 | sell | 100    | 5250  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/JAN23"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 17784        | 180000         | 1             |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 34679        | 180000         | 1             | 40          |

    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17018938 | 90000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2801062 | 17018938 | 22500 |

  Scenario: An LP with ask orders outside valid range during auction is penalised (0044-LIME-099)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 20000000   |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |
      | party3 | USD   | 1000000    |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/JAN23 | 180000            | 0.02 | submission |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1    | ETH/JAN23 | buy  | 100    | 4750  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/JAN23 | sell | 100    | 5260  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 10     | 5100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/JAN23"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 5000  | 1    | party2 |

    And the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 5000       | TRADING_MODE_CONTINUOUS | 3600    | 4865      | 5139      | 17784        | 180000         | 1             |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 4850  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/JAN23 | buy  | 10     | 4900  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/JAN23" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 5000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 34679        | 180000         | 1             | 40          |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/JAN23 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/JAN23 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" epochs
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2806398 | 17013602 | 90000 |

    When the network moves ahead "2" epochs
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/JAN23"
    Then the parties should have the following account balances:
      | party | asset | market id | margin  | general  | bond  |
      | lp1   | USD   | ETH/JAN23 | 2806398 | 17013602 | 22500 |

  @Now
  Scenario: 001b: lp1 and lp2 under supplies liquidity (and expects to get penalty for not meeting the SLA) since both have orders outside price range, this time with half the fees paid based on score.
    # half of the fees get paid based on score, the other half will be paid based on ELS.
    Given the following network parameters are set:
      | name                                        | value |
      | market.liquidity.equityLikeShareFeeFraction | 0.5   |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 200000 |
      | lp2    | USD   | 15000  |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |
      | party3 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR22 | 80000             | 0.02 | submission |
      | lp_2 | lp2   | ETH/MAR22 | 500               | 0.01 | submission |

    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 2      | 200    | lp-b-1    |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 2      | 200    | lp-s-1    |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 2      | 200    | lp-b-2    |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 2      | 200    | lp-s-2    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 80500          | 1             |
    # # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569036

    And the liquidity fee factor should be "0.02" for the market "ETH/MAR22"

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 10671  | 109329  | 80000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 3829    | 500   |
    #margin_intial lp2: 2*1000*3.5569036*1.5=10671
    #lp1: 21342+98658+80000=200000; lp2: 10671+3829+500=15000

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 7113         | 80500          | 2             |

    Then the network moves ahead "6" blocks

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 10671  | 109329  | 40000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 3829    | 250   |
    #liquidity fee: 1000*0.02 = 20; lp1 get 19, lp2 get 0

    Then the following transfers should happen:
      | from   | to     | from account                   | to account                     | market id | amount | asset |
      | market | lp1    | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 14     | USD   |
      | lp1    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/MAR22 | 40000  | USD   |
      | lp2    | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE         | ETH/MAR22 | 250    | USD   |
      | lp1    | market | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_INSURANCE         | ETH/MAR22 | 14     | USD   |

    And the insurance pool balance should be "40269" for the market "ETH/MAR22"

    Then the network moves ahead "6" blocks

    And the parties should have the following account balances:
      | party | asset | market id | margin | general | bond  |
      | lp1   | USD   | ETH/MAR22 | 10671  | 109329  | 20000 |
      | lp2   | USD   | ETH/MAR22 | 10671  | 3829    | 125   |

    And the insurance pool balance should be "60394" for the market "ETH/MAR22"
  # #increament in insurancepool: 60394-40269=20125 which is coming from SLA penalty on lp1 and lp2


