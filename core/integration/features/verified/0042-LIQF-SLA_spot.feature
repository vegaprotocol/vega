Feature: Calculating SLA Performance

  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | default-log-normal-risk-model | 1                | default-none | default-none     | default-basic |
    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |

    And the average block duration is "1"

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount              |
      | lp1    | BTC   | 1000000000000000000 |
      | lp2    | BTC   | 1000000000000000000 |
      | lp3    | BTC   | 1000000000000000000 |
      | lp4    | BTC   | 1000000000000000000 |
      | aux1   | BTC   | 1000000000000000000 |
      | aux2   | BTC   | 1000000000000000000 |
      | party1 | BTC   | 1000000000000000000 |
      | party2 | BTC   | 1000000000000000000 |
      | lp1    | ETH   | 1000000000000000000 |
      | lp2    | ETH   | 1000000000000000000 |
      | lp3    | ETH   | 1000000000000000000 |
      | lp4    | ETH   | 1000000000000000000 |
      | aux1   | ETH   | 1000000000000000000 |
      | aux2   | ETH   | 1000000000000000000 |
      | party1 | ETH   | 1000000000000000000 |
      | party2 | ETH   | 1000000000000000000 |

  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is non-zero. (0042-LIQF-085)(0042-LIQF-101)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 0.5                    |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.25 fees are penalised (then returned as a bonus as they are the only LP)
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 75     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 25     | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 25     | ETH   |

  Scenario: LP fulfills the mininum time fraction and provides liquidity throughout the epoch when performance hysteresis epochs is 1. (0042-LIQF-086)(0042-LIQF-101)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 1                            | 1                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=1.00, then check ~0.0 fees are penalised
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account           | market id | amount | asset |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL | BTC/ETH   | 100    | ETH   |

  Scenario: LPs previous failures to meet the minimum time fraction if the markets performance hysteresis epochs is increased. (0042-LIQF-101)(0042-LIQF-088)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 1.0                          | 0                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |

    # Ensure LPs average timeBookFraction ~0.75 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |

    # Update the market by increasing 'performance hysteresis epochs'. This will take effect for next epoch and
    # no penalty should be applied.
    Given the liquidity sla params named "updated-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 2                             | 1                      |
    And the spot markets are updated:
      | id      | sla params         | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-sla-params | 1e-3                   | 0                         |

    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 100    | ETH   |

  Scenario: LP does not fulfill the mininum time fraction when performance hysteresis epochs is 1. (0042-LIQF-101)(0042-LIQF-087)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 1                            | 1                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction <1.00, then check ~1.0 fees are penalised
    Given the network moves ahead "1" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 100    | ETH   |

  Scenario: LPs average penalty over the last N epochs is worse then their current performance when performance hysteresis epochs is > 1. (0042-LIQF-101)(0042-LIQF-089)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.0                          | 3                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Ensure LPs average timeBookFraction ~0.75 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 75     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 25     | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 25     | ETH   |

  Scenario: LPs average penalty over the last N epochs is worse then their current performance when performance hysteresis epochs is > 1. (0042-LIQF-090)(0042-LIQF-101)(0042-LIQF-104)(0042-LIQF-091)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.0                          | 3                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Ensure LPs average timeBookFraction ~0.50 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "30" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "30" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 50     | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |

  Scenario: 2 LPs fulfill the mininum time fraction but have different SLA performance when the sla competition factor is non-zero.(0042-LIQF-101)(0042-LIQF-102)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 3                             | 1                      |
    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 1m58s |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 2 LPs who are initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
      | lp2 | lp2   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
      | lp2   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp2-ice-buy-1  |
      | lp2   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp2-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "200" for the market "BTC/ETH"

    # Ensure LP1s average timeBookFraction ~0.75 and penalties will be ~0.5 in the next epoch
    Given the network moves ahead "90" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    # Ensure LP2s average timeBookFraction ~0.875 and penalties will be ~0.75 in the next epoch
    Given the network moves ahead "15" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp2   | lp2-ice-buy-1  |
      | lp2   | lp2-ice-sell-1 |
    And the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      |      | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 50     | ETH   |
      | lp2  | lp2 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 75     | ETH   |
      | lp2  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 25     | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 30     | ETH   |
      |      | lp2 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 45     | ETH   |

  Scenario: 2 LPs fulfill the mininum time fraction but have different SLA performance when the sla competition factor is non-zero. (0042-LIQF-101)(0042-LIQF-103)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 3                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 1m58s |

    # Setup the market with 1 LP who is initially meeting their commitment and 1 LP who isn't
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
      | lp2 | lp2   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
      | lp2   | BTC/ETH   | 9         | 9                    | buy  | BID              | 1      | 1      | lp1-ice-buy-1  |
      | lp2   | BTC/ETH   | 9         | 9                    | sell | ASK              | 1      | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "200" for the market "BTC/ETH"

    # Ensure LP1s average timeBookFraction ~0.75 and penalties will be ~0.5 in the next epoch
    Given the network moves ahead "90" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      |      | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 50     | ETH   |
      | lp2  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 100    | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 150    | ETH   |


  Scenario: 4 LPs acheive various penalty fractions, unpaid liquidity fees distributed correctly as a bonus (0042-LIQF-105)
    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0                            | 1                             | 1                      |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 98s   |
      | market.liquidity.equityLikeShareFeeFraction | 1   |


    # Setup the market with 4 LPs who initially meet their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000000          | 0.1 | submission |
      | lp2 | lp2   | BTC/ETH   | 1000000           | 0.1 | submission |
      | lp3 | lp3   | BTC/ETH   | 70000000          | 0.1 | submission |
      | lp4 | lp4   | BTC/ETH   | 919000000         | 0.1 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | BTC/ETH   | buy  | 10000  | 999   | 0                | TYPE_LIMIT | TIF_GTC | lp1-bid   |
      | lp1   | BTC/ETH   | sell | 10000  | 1001  | 0                | TYPE_LIMIT | TIF_GTC | lp1-ask   |
      | lp2   | BTC/ETH   | buy  | 1000   | 999   | 0                | TYPE_LIMIT | TIF_GTC | lp2-bid   |
      | lp2   | BTC/ETH   | sell | 1000   | 1001  | 0                | TYPE_LIMIT | TIF_GTC | lp2-ask   |
      | lp3   | BTC/ETH   | buy  | 70000  | 999   | 0                | TYPE_LIMIT | TIF_GTC | lp3-bid   |
      | lp3   | BTC/ETH   | sell | 70000  | 1001  | 0                | TYPE_LIMIT | TIF_GTC | lp3-ask   |
      | lp4   | BTC/ETH   | buy  | 919000 | 999   | 0                | TYPE_LIMIT | TIF_GTC | lp4-bid   |
      | lp4   | BTC/ETH   | sell | 919000 | 1001  | 0                | TYPE_LIMIT | TIF_GTC | lp4-ask   |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the network moves ahead "1" epochs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1000   | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100000" for the market "BTC/ETH"

    # Ensure LPs have the correct penalty fractions whilst having approx. equal average liquidity scores
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp4   | lp4-bid   | 999   | -1         | TIF_GTC |
      | lp4   | lp4-ask   | 1001  | -1         | TIF_GTC |
    And the network moves ahead "40" blocks
    And the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp3   | lp3-bid   | 999   | -1         | TIF_GTC |
      | lp3   | lp3-ask   | 1001  | -1         | TIF_GTC |
    And the network moves ahead "55" blocks
    And the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp2   | lp2-bid   | 999   | -1         | TIF_GTC |
      | lp2   | lp2-ask   | 1001  | -1         | TIF_GTC |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 1000   | ETH   |
      |      | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 100    | ETH   |
      |      | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 7000   | ETH   |
      |      | lp4 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 91900  | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 1000   | ETH   |
      | lp2  | lp2 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 95     | ETH   |
      | lp3  | lp3 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 2800   | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 24673  | ETH   |
      |      | lp2 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 2344   | ETH   |
      |      | lp3 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 69087  | ETH   |

  Scenario: LP fulfills the mininum time fraction but only provides liquidity scattered throughout the epoch (0042-LIQF-083)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 1.0                    |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.5 fees are penalised (then returned as a bonus as they are the only LP)
    Given the network moves ahead "10" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "5" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "10" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "5" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |
    And the network moves ahead "10" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-3  |
      | lp1   | lp1-ice-sell-3 |
    And the network moves ahead "5" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-4  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-4 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 50     | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |

  Scenario: LP fulfills the mininum time fraction but only provides liquidity at the start of the epoch (0042-LIQF-082)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 1.0                    |
    And the spot markets are updated:
      | id      | sla params          | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | BTC/ETH   | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/ETH   | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/ETH   | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.5 fees are penalised (then returned as a bonus as they are the only LP)
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | BTC/ETH   | 100    | ETH   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | BTC/ETH   | 50     | ETH   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | BTC/ETH   | 50     | ETH   |
