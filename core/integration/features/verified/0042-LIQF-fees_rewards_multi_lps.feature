Feature: Test liquidity provider reward distribution when there are multiple liquidity providers;

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
      | USD | 4              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |
    And the following network parameters are set:
      | name                                             | value |
      | market.value.windowLength                        | 1h    |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | limits.markets.maxPeggedOrders                   | 8     |
      | market.liquidity.providersFeeCalculationTimeStep | 10s   |
      | market.liquidity.equityLikeShareFeeFraction      | 1     |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 1.0              | 24h         | 1.0            |
    
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e6                    | 1e6                       | SLA        |

    Given the average block duration is "2"

  @Now
  Scenario: 001: All liquidity providers in the market receive a greater than zero amount of liquidity fee
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount         |
      | lp1    | USD   | 10000000000000 |
      | lp2    | USD   | 10000000000000 |
      | lp3    | USD   | 10000000000000 |
      | party1 | USD   | 1000000000000  |
      | party2 | USD   | 100000000000   |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 1000000           | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 1000000           | 0.002 | amendment  |
      | lp3 | lp3   | ETH/MAR22 | 1000000000        | 0.003 | submission |
      | lp3 | lp3   | ETH/MAR22 | 1000000000        | 0.003 | amendment  |
    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 38        | 1                    | buy  | BID              | 38     | 20     |
      | lp1   | ETH/MAR22 | 31        | 1                    | sell | ASK              | 31     | 20     |
      | lp2   | ETH/MAR22 | 38        | 1                    | buy  | BID              | 38     | 20     |
      | lp2   | ETH/MAR22 | 31        | 1                    | sell | ASK              | 31     | 20     |
      | lp3   | ETH/MAR22 | 38        | 1                    | buy  | BID              | 38     | 20     |
      | lp3   | ETH/MAR22 | 31        | 1                    | sell | ASK              | 31     | 20     |
 
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 355690000    | 1001010000     | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569036*10000

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 880   | 114    |
      | buy  | 900   | 10     |
      | sell | 1100  | 10     |
      | sell | 1120  | 93     |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0000099899101907 | 10000                   |
      | lp2   | 0.0009989910190707 | 1010000                 |
      | lp3   | 0.9989910190707386 | 1001010000              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin     | general       | bond       |
      | lp1    | USD   | ETH/MAR22 | 1653960171 | 9998346029829 | 10000      |
      | lp2    | USD   | ETH/MAR22 | 1653960171 | 9998345039829 | 1000000    |
      | lp3    | USD   | ETH/MAR22 | 1653960171 | 9997346039829 | 1000000000 |
      | party1 | USD   | ETH/MAR22 | 228207540  | 999771792460  |            |
      | party2 | USD   | ETH/MAR22 | 1120424632 | 98879575368   |            |

    Then the network moves ahead "1" blocks

    And the liquidity fee factor should be "0.003" for the market "ETH/MAR22"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    Then the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party2 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 600000 | USD   |

    And the accumulated liquidity fees should be "600000" for the market "ETH/MAR22"

    And the parties should have the following account balances:
      | party  | asset | market id | margin     | general       | bond       |
      | lp1    | USD   | ETH/MAR22 | 1653960171 | 9998346029829 | 10000      |
      | lp2    | USD   | ETH/MAR22 | 1653960171 | 9998345039829 | 1000000    |
      | lp3    | USD   | ETH/MAR22 | 1653960171 | 9997346039829 | 1000000000 |
      | party1 | USD   | ETH/MAR22 | 1067071078 | 998933008922  |            |
      | party2 | USD   | ETH/MAR22 | 1120424632 | 98878695368   |            |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 880   | 114    |
      | buy  | 900   | 10     |
      | sell | 1100  | 10     |
      | sell | 1120  | 93     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "600000" for the market "ETH/MAR22"
    Then the network moves ahead "11" blocks


    And the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 5      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 599    | USD   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 599394 | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

# # liquidity fee lp1 = 600000 * 0.0000099899101907 = 5
# # liquidity fee lp2 = 600000 * 0.0009989910190707 = 599
# # liquidity fee lp3 = 600000 * 0.9989910190707386 = 599396


