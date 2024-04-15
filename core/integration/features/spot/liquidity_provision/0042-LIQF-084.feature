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
  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is zero. (0042-LIQF-084)

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 500               | 0.1 | submission |
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
    # Leave opening auction, then nothing happens for 31 blocks (getting us past 50% of the time fraction)
    And the network moves ahead "45" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "101" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.0 fees are penalised
    # 50 blocks in to the 58s (or 58 block) long epoch.
    When the network moves ahead "5" blocks
    Then the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |

    When the network moves ahead "1" epochs
    # We should see these transfers
    Then the following transfers should happen:
     | from | to  | from account                   | to account                     | market id | amount | asset |
     |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 101    | ETH   |
     | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 101    | ETH   |

  @SPOTSLA
  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is zero. Same as previous scenario, different timings. (0042-LIQF-084)

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 500               | 0.1 | submission |
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
    # Leave opening auction, then nothing happens for 31 blocks (getting us past 50% of the time fraction)
    And the network moves ahead "3" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "101" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.0 fees are penalised
    When the network moves ahead "45" blocks
    Then the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |

    When the network moves ahead "1" epochs
    # We should see these transfers
    Then the following transfers should happen:
     | from | to  | from account                   | to account                     | market id | amount | asset |
     |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 101    | ETH   |
     | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 101    | ETH   |

  @SPOTSLA
  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is zero. (0042-LIQF-085)

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 500               | 0.1 | submission |
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
    # Leave opening auction, then nothing happens for 31 blocks (getting us past 50% of the time fraction)
    And the network moves ahead "45" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "101" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.0 fees are penalised
    # 50 blocks in to the 58s (or 58 block) long epoch.
    When the network moves ahead "5" blocks
    Then the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |

    When the network moves ahead "1" epochs
    # We should see these transfers
    Then the following transfers should happen:
     | from | to  | from account                   | to account                     | market id | amount | asset |
     |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 101    | ETH   |
     | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 101    | ETH   |

  @SPOTSLA
  Scenario: LP fulfills the mininum time fraction but is not always on the book when the sla competition factor is zero. Same as previous scenario, different timings. The next epoch they meet their commitment such that time on book is 0.75, penalty should be 0.25. (0042-LIQF-085)

    # Setup the market with 1 LP who is initially meeting their commitment
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 500               | 0.1 | submission |
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
    # Leave opening auction, then nothing happens for 31 blocks (getting us past 50% of the time fraction)
    And the network moves ahead "3" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # Generate liquidity fees to be allocated to the LP
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "101" for the market "BTC/ETH"

    # Ensure LPs timeBookFraction ~=0.75, then check ~0.0 fees are penalised
    When the network moves ahead "44" blocks
    Then the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-1  |
      | lp1   | lp1-ice-sell-1 |
    # re-submit the orders to meet commitment amount
    When the network moves ahead "1" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference      |
      | lp1   | BTC/ETH   | 4         | 1                    | buy  | MID              | 4      | 1      | lp1-ice-buy-2  |
      | lp1   | BTC/ETH   | 4         | 1                    | sell | MID              | 4      | 1      | lp1-ice-sell-2 |

    When the network moves ahead "1" epochs
    # We should see these transfers
    Then the following transfers should happen:
     | from | to  | from account                   | to account                     | market id | amount | asset |
     |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 101    | ETH   |
     | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 101    | ETH   |

    # Now perform some trade so that there are fees to go around once again
    When the network moves ahead "40" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "101" for the market "BTC/ETH"
    # Move forwards another 4 blocks to get to 75% of time on book (58 block epochs, 75% == 43.5 blocks)
    When the network moves ahead "4" blocks
    Then the parties cancel the following orders:
      | party | reference      |
      | lp1   | lp1-ice-buy-2  |
      | lp1   | lp1-ice-sell-2 |
    When the network moves ahead "1" epochs
    # We should see these transfers
    Then debug transfers
    Then the following transfers should happen:
     | from | to  | from account                   | to account                     | market id | amount | asset |
     |      | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY    | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 101    | ETH   |
     | lp1  | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL           | BTC/ETH   | 101    | ETH   |
