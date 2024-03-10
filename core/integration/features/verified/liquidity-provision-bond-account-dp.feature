Feature: Check that bond slashing works with non-default asset decimals, market decimals, position decimals.

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 3              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the following network parameters are set:
      | name                                          | value |
      | market.liquidity.bondPenaltyParameter       | 0.1   |
      | limits.markets.maxPeggedOrders                | 2     |
      | validators.epoch.length                       | 5s    |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.24             | 24h         | 1.0            |
    
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 2                       | 0.05                   | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 2700000   |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | party4 | USD   | 100000000 |
      | party5 | USD   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the average block duration is "1"

  @Now
  Scenario: Bond slashing on LP (0044-LIME-002, 0035-LIQM-004, 0044-LIME-009 )

    Given the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party0 | ETH/MAR22 | 500000            | 0   | submission |
      | lp1 | party0 | ETH/MAR22 | 500000            | 0   | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | party0 | ETH/MAR22 | 49        | 1                    | sell | ASK              | 500        | 20     |
      | party0 | ETH/MAR22 | 52        | 1                    | buy  | BID              | 500        | 20     |
 
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party4 | ETH/MAR22 | buy  | 100    | 850   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party5 | ETH/MAR22 | sell | 100    | 1200  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-5 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 500000         | 10            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1030  | 49     |
      | sell | 1010  | 1      |
      | buy  | 990   | 1      |
      | buy  | 970   | 52     |
      | buy  | 900   | 1      |

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general  | bond   |
      | party0 | USD   | ETH/MAR22 | 2134142 | 65858    | 500000 |
      | party1 | USD   | ETH/MAR22 | 11425   | 99988575 |        |
      | party2 | USD   | ETH/MAR22 | 51690   | 99948310 |        |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/MAR22 | 1778452     |
      | party1 | ETH/MAR22 | 10109       |
      | party2 | ETH/MAR22 | 43183       |
    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000 | TRADING_MODE_CONTINUOUS | 1 | 1000 | 1000 | 142276 | 500000 | 40 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general  | bond   |
      | party0 | USD   | ETH/MAR22 | 2134142 | 65858    | 500000 |
      | party1 | USD   | ETH/MAR22 | 11425   | 99988575 |        |
      | party2 | USD   | ETH/MAR22 | 265234  | 99734616 |        |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party0 | ETH/MAR22 | sell | 110    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party0-sell-3 |
      | party1 | ETH/MAR22 | buy  | 110    | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy-4  |

    # extra margin for party0
    # bond slashed
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 533535       | 500000         | 150           |

    And the insurance pool balance should be "40365" for the market "ETH/MAR22"

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general  | bond   |
      | party0 | USD   | ETH/MAR22 | 2603654 | 360      | 55981  |
      | party1 | USD   | ETH/MAR22 | 117826  | 99881624 |        |
      | party2 | USD   | ETH/MAR22 | 265234  | 99734696 |        |
      | party3 | USD   | ETH/MAR22 | 28826   | 99971294 |        |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/MAR22 | 2169712     |
      | party1 | ETH/MAR22 | 98189       |
      | party2 | ETH/MAR22 | 221029      |
