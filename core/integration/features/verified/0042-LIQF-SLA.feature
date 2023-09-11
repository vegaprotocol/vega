Feature: Calculating SLA Performance

  Background:
    
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params    |
      | ETH/DEC23 | ETH        | USD   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-basic |
    And the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 2     |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | validators.epoch.length                          | 58s   |
      | market.liquidity.stakeToCcyVolume                | 1     |
    And the average block duration is "1"

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | aux1   | USD   | 1000000000 |
      | aux2   | USD   | 1000000000 |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |


  Scenario: LP fulfills the mininum time fraction but only provides liquidity at the start of the epoch (0042-LIQF-037)(0042-LIQF-043)(0042-LIQF-046)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 1.0                    |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.5 fees are penalised (then returned as a bonus as they are the only LP)
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 50     | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |


  Scenario: LP fulfills the mininum time fraction but only provides liquidity scattered throughout the epoch (0042-LIQF-038)(0042-LIQF-043)(0042-LIQF-046)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 1.0                    |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.5 fees are penalised (then returned as a bonus as they are the only LP)
    Given the network moves ahead "10" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "5" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "10" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "5" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |
    And the network moves ahead "10" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-3  |
      | lp1   | lp1-ice-sell-3 |
    And the network moves ahead "5" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-4  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-4 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 50     | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |


  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is zero. (0042-LIQF-041)(0042-LIQF-043)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 0                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.0 fees are penalised
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | ETH/DEC23 | 100    | USD   |


  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is non-zero. (0042-LIQF-042)(0042-LIQF-043)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 0                             | 0.5                    |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.25 fees are penalised (then returned as a bonus as they are the only LP)
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 75     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 25     | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 25     | USD   |


  Scenario: LP fulfills the mininum time fraction and provides liquidity throughout the epoch when performance hysteresis epochs is 1. (0042-LIQF-035)(0042-LIQF-043)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 1                            | 1                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Ensure LPs timeBookFraction ~=1.00, then check ~0.0 fees are penalised
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account           | market id | amount | asset |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL | ETH/DEC23 | 100    | USD   |


  Scenario: LP does not fulfill the mininum time fraction when performance hysteresis epochs is 1. (0042-LIQF-043)(0042-LIQF-049)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 1                            | 1                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Ensure LPs timeBookFraction <1.00, then check ~1.0 fees are penalised
    Given the network moves ahead "1" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC23 | 100    | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_INSURANCE         | ETH/DEC23 | 100    | USD   |


  Scenario: LPs previous failures to meet the minimum time fraction if the markets performance hysteresis epochs is increased. (0042-LIQF-043)(0042-LIQF-053)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 1.0                          | 0                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |

    # Ensure LPs average timeBookFraction ~0.75 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    # Update the market by increasing 'performance hysteresis epochs'. No penalty should be applied.
    Given the liquidity sla params named "updated-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 2                             | 1                      |
    And the markets are updated:
      | id        | sla params         | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | updated-sla-params | 1e-3                   | 0                         |
    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | ETH/DEC23 | 100    | USD   |


  Scenario: LPs average penalty over the last N epochs is worse then their current performance when performance hysteresis epochs is > 1. (0042-LIQF-043)(0042-LIQF-047)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.0                          | 3                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Ensure LPs average timeBookFraction ~0.75 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "45" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 75     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 25     | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 25     | USD   |


  Scenario: LPs average penalty over the last N epochs is worse then their current performance when performance hysteresis epochs is > 1. (0042-LIQF-039)(0042-LIQF-043)(0042-LIQF-046)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.0                          | 3                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Ensure LPs average timeBookFraction ~0.50 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "30" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "30" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-3  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-3 |

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 50     | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |


  Scenario: LPs average penalty over the last N epochs is better then their current performance when performance hysteresis epochs is > 1. (0042-LIQF-040)(0042-LIQF-043)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.0                          | 3                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Ensure LPs average timeBookFraction ~0.50 over the last 2 epochs, and will be ~1.0 in the next epoch
    Given the network moves ahead "30" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-2  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-2 |
    And the network moves ahead "30" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    And the network moves ahead "1" epochs

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "100" for the market "ETH/DEC23"

    When the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                   | to account                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC23 | 100    | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_INSURANCE         | ETH/DEC23 | 100    | USD   |


  Scenario: 2 LPs fulfill the mininum time fraction but have different SLA performance when the sla competition factor is non-zero.(0042-LIQF-043)(0042-LIQF-044)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 3                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 1m58s |

    # Setup the market with 2 LPs who are initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
      | lp2 | lp2   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
      | lp2   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp2-ice-buy-1  |
      | lp2   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp2-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "200" for the market "ETH/DEC23"

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
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      |      | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 50     | USD   |
      | lp2  | lp2 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 75     | USD   |
      | lp2  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 25     | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 30     | USD   |
      |      | lp2 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 45     | USD   |


  Scenario: 2 LPs fulfill the mininum time fraction but have different SLA performance when the sla competition factor is non-zero. (0042-LIQF-043)(0042-LIQF-045)

    # Initialise the market with the required parameters
    Given the liquidity sla params named "scenario-sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.5                          | 3                             | 1                      |
    And the markets are updated:
      | id        | sla params          | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | scenario-sla-params | 1e-3                   | 0                         |

    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 1m58s |

    # Setup the market with 1 LP who is initially meeting their commitment and 1 LP who isn't
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/DEC23 | 10000             | 0.1 | submission |
      | lp2 | lp2   | ETH/DEC23 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | ETH/DEC23 | 200       | 120                  | buy  | BID              | 1000   | 1      | lp1-ice-buy-1  |
      | lp1   | ETH/DEC23 | 200       | 120                  | sell | ASK              | 1000   | 1      | lp1-ice-sell-1 |
      | lp2   | ETH/DEC23 | 9         | 9                    | buy  | BID              | 1      | 1      | lp1-ice-buy-1  |
      | lp2   | ETH/DEC23 | 9         | 9                    | sell | ASK              | 1      | 1      | lp1-ice-sell-1 |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC23 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC23 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC23"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Generate liquidity fees to be allocated to the LP
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC23 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC23 | sell | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "200" for the market "ETH/DEC23"

    # Ensure LP1s average timeBookFraction ~0.75 and penalties will be ~0.5 in the next epoch
    Given the network moves ahead "90" blocks
    And the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    And the network moves ahead "1" epochs
    Then the following transfers should happen:
      | from | to  | from account                                   | to account                                     | market id | amount | asset |
      |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      |      | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY                    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ETH/DEC23 | 100    | USD   |
      | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 50     | USD   |
      | lp1  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 50     | USD   |
      | lp2  |     | ACCOUNT_TYPE_LP_LIQUIDITY_FEES                 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ETH/DEC23 | 100    | USD   |
      |      | lp1 | ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION | ACCOUNT_TYPE_GENERAL                           | ETH/DEC23 | 150    | USD   |