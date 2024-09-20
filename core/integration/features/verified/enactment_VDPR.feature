Feature: Volume discount program - program enactment

  Volume discount program rewards parties who comprise above a specified
  taker volume with a discount on their fees.

  Tests check on program enactment party benefit factors are updated
  correctly and applied in the next epoch.

  Background:
    
    # Initialise the network and register the assets
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.01  |
      | market.fee.factors.infrastructureFee    | 0.01  |
      | market.fee.factors.treasuryFee          | 0.1   |
      | market.fee.factors.buybackFee           | 0.1   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | validators.epoch.length                 | 20s   |
      | market.auction.minimumDuration          | 1     |

    And the following assets are registered:
      | id      | decimal places | quantum |
      | USD-0-1 | 0              | 1       |
      | MXN-0-1 | 0              | 1       |

    # Initialise the parties and deposit assets
    Given the parties deposit on asset's general account the following amount:
      | party | asset   | amount   |
      | aux1  | USD-0-1 | 1000000  |
      | aux2  | USD-0-1 | 1000000  |
      | aux1  | MXN-0-1 | 10000000 |
      | aux2  | MXN-0-1 | 10000000 |

    # Setup the markets
    And the markets:
      | id          | quote name | asset   | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | BTC/USD-0-1 | USD        | USD-0-1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | BTC/MXN-0-1 | VND        | MXN-0-1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the spot markets:
      | id              | name    | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places | position decimal places | sla params    |
      | MXN-0-1/USD-0-1 | MXN/USD | MXN-0-1    | USD-0-1     | default-log-normal-risk-model | 1                | default-none | default-none     | 0              | 0                       | default-basic |
      | USD-0-1/MXN-0-1 | MXN/USD | USD-0-1    | MXN-0-1     | default-log-normal-risk-model | 1                | default-none | default-none     | 0              | 0                       | default-basic |
    And the parties place the following orders:
      | party | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USD-0-1 | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id   | side | volume | price  | resulting trades | type       | tif     |
      | aux1  | BTC/MXN-0-1 | buy  | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/MXN-0-1 | sell | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id       | side | volume | price | resulting trades | type       | tif     |
      | aux1  | MXN-0-1/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | MXN-0-1/USD-0-1 | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id       | side | volume | price  | resulting trades | type       | tif     |
      | aux1  | USD-0-1/MXN-0-1 | buy  | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | USD-0-1/MXN-0-1 | sell | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USD-0-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "MXN-0-1/USD-0-1"

    Given the parties deposit on asset's general account the following amount:
      | party  | asset   | amount   |
      | party1 | USD-0-1 | 1000000  |
      | party2 | USD-0-1 | 1000000  |
      | party1 | MXN-0-1 | 10000000 |
      | party2 | MXN-0-1 | 10000000 |
      

  Scenario: No program currently active, new program enacted, discount factors applied from the start of the next epoch.

    # First generate some taker volume, so in the next epoch after the program is created party1 will qualify for discount factors
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | aux1   | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party1 | <market>  | buy  | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | buyer infrastructure fee |
      | party1 | aux1   | 1    | 50000 | buy            | 500             | 500                      |

    # Enact the new volume discount program
    Given the volume discount program tiers named "vdt":
      | volume | infra factor | liquidity factor | maker factor |
      | 1      | 0.1          | 0.1              | 0.1          |
    And the volume discount program:
      | id  | tiers | closing timestamp | window length |
      | id1 | vdt   | 0                 | 1             |

    # Move ahead to the first epoch after enactment, check party1 is receiving discounts proportional to the discount factors
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | aux1   | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party1 | <market>  | buy  | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | buyer infrastructure fee | buyer maker fee volume discount | buyer infrastructure fee volume discount |
      | party1 | aux1   | 1    | 50000 | buy            | 450             | 450                      | 50                              | 50                                       |

  Examples:
    # Check the above scenario for a derivative and spot market    
      | market          |
      | BTC/USD-0-1     |
      | MXN-0-1/USD-0-1 |


  Scenario: Program currently active, program update enacted, discount factors applied from the start of the next epoch.

    # Enact the original referral program and move to the first epoch after enactment
    Given the volume discount program tiers named "vdt":
      | volume | infra factor | liquidity factor | maker factor |
      | 1      | 0.1          | 0.1              | 0.1          |
    And the volume discount program:
      | id  | tiers | closing timestamp | window length |
      | id1 | vdt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    # First generate some taker volume, so in the next epoch party1 will qualify for the discount factors
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | aux1   | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party1 | <market>  | buy  | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | buyer infrastructure fee |
      | party1 | aux1   | 1    | 50000 | buy            | 500             | 500                      |

    # Move ahead an epoch so factors updated, check party1 is receiving discounts proportional to the discount factors
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | aux1   | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party1 | <market>  | buy  | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | buyer infrastructure fee | buyer maker fee volume discount | buyer infrastructure fee volume discount |
      | party1 | aux1   | 1    | 50000 | buy            | 450             | 450                      | 50                              | 50                                       |

    # Enact an update to the discount program - doubling the discounts
    Given the volume discount program tiers named "vdt":
      | volume | infra factor | liquidity factor | maker factor |
      | 1000   | 0.2          | 0.2              | 0.2          |
    And the volume discount program:
      | id  | tiers | closing timestamp | window length |
      | id1 | vdt   | 0                 | 1             |

    # Before moving to the next epoch, check party1 is still receiving discounts proportional to the original discount factor
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | aux1   | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party1 | <market>  | buy  | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | buyer infrastructure fee | buyer maker fee volume discount | buyer infrastructure fee volume discount |
      | party1 | aux1   | 1    | 50000 | buy            | 450             | 450                      | 50                              | 50                                       |

    # # Move to the first epoch after program update, check party1 is now receiving discounts proportional to the updated discount factor
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | aux1   | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party1 | <market>  | buy  | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then debug trades
    And the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | buyer infrastructure fee | buyer maker fee volume discount | buyer infrastructure fee volume discount |
      | party1 | aux1   | 1    | 50000 | buy            | 400             | 400                      | 100                             | 100                                      |

  Examples:
    # Check the above scenario for a derivative and spot market    
      | market          |
      | BTC/USD-0-1     |
      | MXN-0-1/USD-0-1 |