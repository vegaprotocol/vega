Feature: Calculating SLA Performance

  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.0              | 10s         | 0.75           |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 0                             | 0                      |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model          | auction duration | fees          | price monitoring | sla params | liquidity monitoring |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | simple-risk-model-1 | 2                | fees-config-1 | price-monitoring | SLA        | lqm-params           |

    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |

    And the average block duration is "1"

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount              |
      | lp1    | ETH   | 1000000000000000000 |
      | lp1    | BTC   | 1000000000000000000 |
      | aux1   | ETH   | 1000000000000000000 |
      | aux1   | BTC   | 1000000000000000000 |
      | aux2   | ETH   | 1000000000000000000 |
      | aux2   | BTC   | 1000000000000000000 |
      | party1 | ETH   | 1000000000000000000 |
      | party1 | BTC   | 1000000000000000000 |
      | party2 | BTC   | 1000000000000000000 |

  @SPOTSLA
  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is zero. (0042-LIQF-041)(0042-LIQF-043)

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 5000              | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 4         | 1                    | buy  | MID              | 4      | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 4         | 1                    | sell | MID              | 4      | 1      | lp1-ice-sell-1 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "3" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "101" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.0 fees are penalised
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    #When the network moves ahead "9" blocks
    Then debug transfers
    And debug trades
    # We should see these transfers
    #Then the following transfers should happen:
    # | from | to  | from account                   | to account                     | market id | amount | asset |
    # |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 100    | BTC   |
    # | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 100    | BTC   |
    # But instead get this:
    Then the following transfers should happen:
      | from | to | from account                   | to account                    | market id | amount | asset | type                                   |
      | lp1  |    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 101    | ETH   | TRANSFER_TYPE_SLA_PENALTY_LP_FEE_APPLY |
