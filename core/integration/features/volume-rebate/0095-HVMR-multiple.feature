Feature: Volume rebate program - two programs overlapping
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
      | id       | decimal places | quantum |
      | USD-0-1  | 0              | 1       |
      | MXN-0-10 | 0              | 10      |

    # Initialise the parties and deposit assets
    And the parties deposit on asset's general account the following amount:
      | party  | asset    | amount   |
      | aux1   | USD-0-1  | 3000000  |
      | aux2   | USD-0-1  | 3000000  |
      | aux1   | MXN-0-10 | 30000000 |
      | aux2   | MXN-0-10 | 30000000 |
      | party1 | USD-0-1  | 3000000  |
      | party2 | USD-0-1  | 3000000  |
      | party1 | MXN-0-10 | 30000000 |
      | party2 | MXN-0-10 | 30000000 |

    # Setup the markets
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 1                 |
    And the markets:
      | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | BTC/USD-0-1  | USD        | USD-0-1  | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
      | BTC/MXN-0-10 | VND        | MXN-0-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the spot markets:
      | id               | name    | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places | position decimal places | sla params    |
      | MXN-0-10/USD-0-1 | MXN/USD | MXN-0-10   | USD-0-1     | default-log-normal-risk-model | 1                | default-none | price-monitoring | 0              | 0                       | default-basic |
      | USD-0-1/MXN-0-10 | MXN/USD | USD-0-1    | MXN-0-10    | default-log-normal-risk-model | 1                | default-none | price-monitoring | 0              | 0                       | default-basic |
    And the parties place the following orders:
      | party | market id   | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USD-0-1 | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id    | side | volume | price  | resulting trades | type       | tif     |
      | aux1  | BTC/MXN-0-10 | buy  | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/MXN-0-10 | sell | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id        | side | volume | price | resulting trades | type       | tif     |
      | aux1  | MXN-0-10/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | MXN-0-10/USD-0-1 | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party | market id        | side | volume | price  | resulting trades | type       | tif     |
      | aux1  | USD-0-1/MXN-0-10 | buy  | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | USD-0-1/MXN-0-10 | sell | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USD-0-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/MXN-0-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "MXN-0-10/USD-0-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "MXN-0-10/USD-0-1"

  Scenario Outline: rebate program A is enacted, during its lifetime program B takes over. Program B closes after program A, so after B closes, no programs should be active. (0995-HVMR-006)(0095-HMVR-007)(0095-HVMR-008)(0095-HVMR-009)(0095-HVM-010)(0095-HVMR-011).
    Given the volume rebate program tiers named "A1":
      | fraction | rebate |
      | 0.0001   | 0.001  |
    And the volume rebate program tiers named "B3":
      | fraction | rebate |
      | 0.0001   | 0.002  |
      | 0.0002   | 0.003  |
      | 0.0003   | 0.004  |
    And the volume rebate program:
      | id       | tiers | closing timestamp | window length   | closing delta   |
      | programA | A1    | 0                 | <window length> | <closing delta> |
    And the network moves ahead "1" epochs

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0               | 500              |

    Given the network moves ahead <epochs between trades> epochs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party2 | aux1   | 1    | 50000 | sell           | 0               | 500              |

    When the network moves ahead "1" epochs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 2      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | <party1 rebate A>            |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | <party2 rebate A>            |

    When the network moves ahead "1" epochs
    Then the volume rebate program:
      | id       | tiers | closing timestamp | window length   | closing delta   |
      | programB | B3    | 0                 | <window length> | <closing delta> |

    When the network moves ahead "1" epochs
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0               | 500              |

    Given the network moves ahead <epochs between trades> epochs
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party2 | aux1   | 1    | 50000 | sell           | 0               | 500              |

    When the network moves ahead "1" epochs
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 2      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | <party1 rebate B>            |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | <party2 rebate B>            |

    # Now no programs are active
    When the network moves ahead "4" epochs
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party2 | aux1   | 1    | 50000 | sell           | 0               | 500              |

    When the network moves ahead "1" epochs
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 2      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 0                            |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | 0                            |

  # Closing delta must be > 1 epoch + epochs betwochs between trades + 3 blocks, 80s == 4 epochs
  Examples:
      | market           | window length | epochs between trades | party1 rebate A | party2 rebate A | closing delta | party1 rebate B | party2 rebate B |
      | BTC/USD-0-1      | 2             | "0"                   | 50              | 50              | 80s           | 200             | 200             |
      | BTC/USD-0-1      | 2             | "1"                   | 50              | 50              | 80s           | 200             | 200             |
      | MXN-0-10/USD-0-1 | 2             | "0"                   | 50              | 50              | 80s           | 200             | 200             |
      | MXN-0-10/USD-0-1 | 2             | "1"                   | 50              | 50              | 80s           | 200             | 200             |

