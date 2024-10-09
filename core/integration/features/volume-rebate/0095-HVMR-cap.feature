Feature: Volume rebate program - rebate cap

  Background:
    
    # Initialise the network and register the assets
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.01  |
      | market.fee.factors.infrastructureFee    | 0.01  |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | validators.epoch.length                 | 20s   |
      | market.auction.minimumDuration          | 1     |

    And the following assets are registered:
      | id      | decimal places | quantum |
      | USD-0-1 | 0              | 1       |

    # Initialise the parties and deposit assets
    Given the parties deposit on asset's general account the following amount:
      | party | asset   | amount  |
      | aux1  | USD-0-1 | 1000000 |
      | aux2  | USD-0-1 | 1000000 |

    # Setup the markets
    Given the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 1                 |
    And the markets:
      | id          | quote name | asset   | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | BTC/USD-0-1 | USD        | USD-0-1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the parties place the following orders:
      | party | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USD-0-1 | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USD-0-1"

    Given the parties deposit on asset's general account the following amount:
      | party  | asset   | amount  |
      | party1 | USD-0-1 | 1000000 |
      | party2 | USD-0-1 | 1000000 |
      

  Scenario Outline: Fixed buyback, treasury and rebate factors. Rebate capped correctly where necessary. (0095-HVMR-029)(0095-HVMR-030)(0095-HVMR-031)

    Given the following network parameters are set:
      | name                           | value      |
      | market.fee.factors.buybackFee  | <buyback>  |
      | market.fee.factors.treasuryFee | <treasury> |
    And the volume rebate program tiers named "vrt":
      | fraction | rebate          |
      | 0.0001   | 0.001           |
      | 0.5000   | <rebate factor> |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 2      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0               | 500              |
      | party2 | aux1   | 2    | 50000 | sell           | 0               | 1000             |

    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 2      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 50                           |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | <rebate amount>              |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount          | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | 50              | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |
      |      | party2 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

  Examples:
      | buyback | treasury | rebate factor | rebate amount |
      | 0.001   | 0.001    | 0.001         | 50            |
      | 0.001   | 0.001    | 0.002         | 100           |
      | 0.001   | 0.001    | 0.003         | 100           |
      | 0.002   | 0.000    | 0.001         | 50            |
      | 0.002   | 0.000    | 0.002         | 100           |
      | 0.002   | 0.000    | 0.003         | 100           |
      | 0.000   | 0.002    | 0.001         | 50            |
      | 0.000   | 0.002    | 0.002         | 100           |
      | 0.000   | 0.002    | 0.003         | 100           |



  Scenario Outline: Fixed buyback and treasury factors. Variable rebate factors. Rebate capped correctly where necessary. (0095-HVMR-032)(0095-HVMR-033)

    Given the following network parameters are set:
      | name                           | value      |
      | market.fee.factors.buybackFee  | <buyback>  |
      | market.fee.factors.treasuryFee | <treasury> |
    And the volume rebate program tiers named "vrt":
      | fraction | rebate                  |
      | 0.0001   | 0.001                   |
      | 0.5000   | <initial rebate factor> |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 2      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0               | 500              |
      | party2 | aux1   | 2    | 50000 | sell           | 0               | 1000             |

    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 3      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 50                           |
      | party2 | aux1   | 2    | 50000 | sell           | 0                           | <initial rebate amount>      |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount                  | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | 50                      | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |
      |      | party2 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <initial rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

    Given the volume rebate program tiers named "vrt":
      | fraction | rebate                  |
      | 0.0001   | 0.001                   |
      | 0.5000   | <updated rebate factor> |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 2             |
    # Note, we are having to forward two epochs here to ensure the
    # program is updated and then the updated benefits are used to set
    # the rebate factor. In future, this may be reworked so the program
    # is upated before the rebates are set on the same epoch boundary.
    When the network moves ahead "2" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 3      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    And the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 50                           |
      | party2 | aux1   | 2    | 50000 | sell           | 0                           | <updated rebate amount>      |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount                  | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | 50                      | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |
      |      | party2 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <updated rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

  Examples:
      | buyback | treasury | initial rebate factor | initial rebate amount | updated rebate factor | updated rebate amount |
      | 0.001   | 0.001    | 0.001                 | 100                   | 0.003                 | 200                   |
      | 0.002   | 0.000    | 0.001                 | 100                   | 0.003                 | 200                   |
      | 0.000   | 0.002    | 0.001                 | 100                   | 0.003                 | 200                   |
      | 0.001   | 0.001    | 0.003                 | 200                   | 0.001                 | 100                   |
      | 0.002   | 0.000    | 0.003                 | 200                   | 0.001                 | 100                   |
      | 0.000   | 0.002    | 0.003                 | 200                   | 0.001                 | 100                   |


  Scenario Outline: Fixed rebate factors. Variable treasury and buyback factors. Rebate capped correctly where necessary. (0095-HVMR-034)(0095-HVMR-035)(0095-HVMR-036)(0095-HVMR-037)(0095-HVMR-038)(0095-HVMR-039)

    Given the following network parameters are set:
      | name                           | value              |
      | market.fee.factors.buybackFee  | <initial buyback>  |
      | market.fee.factors.treasuryFee | <initial treasury> |
    And the volume rebate program tiers named "vrt":
      | fraction | rebate   |
      | 0.0001   | 0.001    |
      | 0.5000   | <rebate> |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 2      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0               | 500              |
      | party2 | aux1   | 2    | 50000 | sell           | 0               | 1000             |

    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 3      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 50                           |
      | party2 | aux1   | 2    | 50000 | sell           | 0                           | <initial rebate amount>      |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount                  | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | 50                      | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |
      |      | party2 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <initial rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

    Given the following network parameters are set:
      | name                           | value              |
      | market.fee.factors.buybackFee  | <updated buyback>  |
      | market.fee.factors.treasuryFee | <updated treasury> |
    When the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 2      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 3      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    And the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 50                           |
      | party2 | aux1   | 2    | 50000 | sell           | 0                           | <updated rebate amount>      |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount                  | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | 50                      | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |
      |      | party2 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <updated rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

  Examples:
      | rebate | initial buyback | initial treasury | initial rebate amount | updated buyback | updated treasury | updated rebate amount |
      | 0.003  | 0.002           | 0.002            | 300                   | 0.001           | 0.001            | 200                   |
      | 0.003  | 0.004           | 0.000            | 300                   | 0.002           | 0                | 200                   |
      | 0.003  | 0.000           | 0.004            | 300                   | 0               | 0.002            | 200                   |
      | 0.003  | 0.001           | 0.001            | 200                   | 0.002           | 0.002            | 300                   |
      | 0.003  | 0.002           | 0.000            | 200                   | 0.004           | 0                | 300                   |
      | 0.003  | 0.000           | 0.002            | 200                   | 0               | 0.004            | 300                   |


  Scenario Outline: Fees updated mid epoch, fees not affected untill next epoch (0029-FEES-051)(0029-FEES-052)

    Given the following network parameters are set:
      | name                           | value              |
      | market.fee.factors.buybackFee  | <initial buyback>  |
      | market.fee.factors.treasuryFee | <initial treasury> |
    And the volume rebate program tiers named "vrt":
      | fraction | rebate   |
      | 0.0001   | <rebate> |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | party2 | 1    | 50000 | sell           | 0               | 500              |

    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | seller treasury fee       | seller buyback fee       | seller high volume maker fee |
      | party1 | party2 | 1    | 50000 | sell           | <initial treasury amount> | <initial buyback amount> | <initial rebate amount>      |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount                  | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <initial rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

    Then clear trade events
    Given the following network parameters are set:
      | name                           | value              |
      | market.fee.factors.buybackFee  | <updated buyback>  |
      | market.fee.factors.treasuryFee | <updated treasury> |
    And the network moves ahead "1" blocks
    When the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then debug trades
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | seller treasury fee       | seller buyback fee       | seller high volume maker fee |
      | party1 | party2 | 1    | 50000 | sell           | <updated treasury amount> | <updated buyback amount> | <updated rebate amount>      |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id   | amount                  | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/USD-0-1 | <updated rebate amount> | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

  Examples:
      | rebate | initial buyback | initial treasury | initial buyback amount | initial treasury amount | initial rebate amount | updated buyback | updated treasury | updated buyback amount | updated treasury amount | updated rebate amount |
      | 0.003  | 0.004           | 0                | 50                     | 0                       | 150                   | 0.001           | 0                | 0                      | 0                       | 50                    |
      | 0.003  | 0               | 0.004            | 0                      | 50                      | 150                   | 0               | 0.001            | 0                      | 0                       | 50                    |
      | 0.003  | 0.002           | 0.002            | 25                     | 25                      | 150                   | 0.001           | 0.001            | 0                      | 0                       | 100                   |

