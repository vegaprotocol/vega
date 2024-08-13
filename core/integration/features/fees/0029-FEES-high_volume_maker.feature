Feature: Discounts from multiple sources

  Background:

    # Initialise timings
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    And the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 15                |

    # Initialise the markets and network parameters
    Given the following network parameters are set:
      | name                                                    | value      |
      | market.fee.factors.infrastructureFee                    | 0.01       |
      | market.fee.factors.makerFee                             | 0.01       |
      | market.auction.minimumDuration                          | 1          |
      | limits.markets.maxPeggedOrders                          | 4          |
      | referralProgram.minStakedVegaTokens                     | 0          |
      | referralProgram.maxPartyNotionalVolumeByQuantumPerEpoch | 1000000000 |
      | referralProgram.maxReferralRewardProportion             | 0.1        |
      | validators.epoch.length                                 | 10s        |

    And the volume rebate program tiers named "vrt":
      | fraction | rebate |
      | 0.2      | 0.001   |

    And the volume rebate program:
      | id  | tiers | closing timestamp | window length |
      | id1 | vrt   | 0                 | 2             |

    And the network moves ahead "1" epochs

    # Initialse the assets and markets
    And the following assets are registered:
      | id  | decimal places | quantum |
      | USD | 1              | 1       |
    And the markets:
      | id      | quote name | asset | risk model            | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD | ETH        | USD   | log-normal-risk-model | margin-calculator-1 | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 1              | 1                       |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 3600s       | 1              |
    When the markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/USD | lqm-params           | 1e-3                   | 0                         |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | lpprov  | USD   | 1000000000 |
      | lpprov2 | USD   | 1000000000 |
      | aux1    | USD   | 1000000000 |
      | aux2    | USD   | 1000000000 |
      | trader1 | USD   | 1000000000 |
      | trader2 | USD   | 1000000000 |
      | trader3 | USD   | 1000000000 |

    # Exit the opening auction
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | lp type    |
      | lp1 | lpprov | ETH/USD   | 1000000           | 0.01 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/USD   | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/USD   | 5000      | 1000                 | sell | ASK              | 10000  | 1      |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD   | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/USD"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD"

  Scenario: When there is `high_volume_market_maker_rebate`, `high_volume_maker_fee` should be taken from the `treasury/buyback_fee` components with value `high_volume_maker_fee = high_volume_factor * trade_value_for_fee_purposes` (0029-FEES-042)

    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/USD   | sell | 210    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/USD   | sell | 310    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/USD   | sell | 480    | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/USD   | buy  | 1000   | 1000  | 3                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" epochs

    Then debug transfers

