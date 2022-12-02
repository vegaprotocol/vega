Feature: check if the realised PnL and unreaslied PnL is calculated according to spec in glossaries: https://github.com/vegaprotocol/specs/blob/master/glossaries/trading-and-protocol-glossary.md

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: 001, check PnL when traders partially close positions, 0007-POSN-011, 0007-POSN-012
    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short: 3.5569036
    #risk factor long: 0.801225765
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1000    | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000000 |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    Then the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 177845       | 5000000        | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1014  | 9975   |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 986   | 10234  |
      | buy  | 900   | 1      |

    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 50     | 0              | 0            |
      | party2 | -50    | 0              | 0            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 5      | 1005  | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-2  |
      | party2 | ETH/MAR22 | buy  | 5      | 1005  | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-2 |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 45     | 225            | 25           |
      | party2 | -45    | -225           | -25          |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 5      | 995   | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-3  |
      | party2 | ETH/MAR22 | buy  | 5      | 995   | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-3 |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 40     | -200           | 0            |
      | party2 | -40    | 200            | 0            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 2      | 1009  | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-4  |
      | party2 | ETH/MAR22 | buy  | 2      | 1009  | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-4 |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 38     | 342            | 18           |
      | party2 | -38    | -342           | -18          |

  Scenario: 002, check PnL at the settlement

    Given time is updated to "2020-11-30T00:00:00Z"

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1000    | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000000 |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    Then the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 177845       | 5000000        | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1014  | 9975   |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 986   | 10234  |
      | buy  | 900   | 1      |

    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 50     | 0              | 0            |
      | party2 | -50    | 0              | 0            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 5      | 1005  | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-2  |
      | party2 | ETH/MAR22 | buy  | 5      | 1005  | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-2 |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 45     | 225            | 25           |
      | party2 | -45    | -225           | -25          |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 5      | 995   | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-3  |
      | party2 | ETH/MAR22 | buy  | 5      | 995   | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-3 |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 40     | -200           | 0            |
      | party2 | -40    | 200            | 0            |

    # party1 and party2 close their position
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 40     | 1009  | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-4  |
      | party2 | ETH/MAR22 | buy  | 40     | 1009  | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-4 |

    # there is no closure of positions, so the realised PnL stays the same
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 360          |
      | party2 | 0      | 0              | -360         |

    # party1 and party2 open their position
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | sell | 5      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | buy-p1-5  |
      | party2 | ETH/MAR22 | buy  | 5      | 1003  | 1                | TYPE_LIMIT | TIF_GTC | sell-p2-5 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1003       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 178378       | 5000000        | 5             |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -5     | 0              | 360          |
      | party2 | 5      | 0              | -360         |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | USD   | ETH/MAR22 | 21534  | 99978826 |
      | party2 | USD   | ETH/MAR22 | 4921   | 99994719 |

    #margin+general-initial = 21534+99978826=360 which is matching with realised pnl for party1
    #margin+general-initial = 4921+99994719=-360 which is matching with realised pnl for party2

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2021-03-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/MAR22"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 800   |

    And then the network moves ahead "10" blocks

    # MTM with the settlement price 800, will add (1003-800)*5=1015 into general account of party1, so new general account will be 99978826+1015=99979841
    # MTM with the settlement price 800, will reduce (1003-800)*5=1015 from general account of party2, so new general account will be 99994719-1015=99993704

    # Check that party positions and overall account balances are the same as before auction start (accounting for a settlement transfer of 200 from party2 to party1)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -5     | 0              | 360          |
      | party2 | 5      | 0              | -360         |

    # all the asset from margin account should be moved to general account:
    # party1: 21534+99979841=100001375
    # party2: 4921+99993704=99998625
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party1 | USD   | ETH/MAR22 | 0      | 100001375 |
      | party2 | USD   | ETH/MAR22 | 0      | 99998625  |
