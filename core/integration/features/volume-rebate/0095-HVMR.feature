Feature: Volume rebate program - contributions from trades

  Volume rebate program rewards parties who comprise above a specified
  fraction of the maker volume on the network in a window with an
  extra rebate factor.

  Tests check trades contribute towards a party's maker volume fraction
  correctly and that volume across windows and markets is correctly 
  counted and scaled where necessary.

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
    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount   |
      | aux1  | USD-0-1  | 1000000  |
      | aux2  | USD-0-1  | 1000000  |
      | aux1  | MXN-0-10 | 10000000 |
      | aux2  | MXN-0-10 | 10000000 |

    # Setup the markets
    Given the price monitoring named "price-monitoring":
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
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USD-0-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/MXN-0-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "MXN-0-10/USD-0-1"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "MXN-0-10/USD-0-1"

    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount   |
      | party1 | USD-0-1  | 1000000  |
      | party2 | USD-0-1  | 1000000  |
      | party1 | MXN-0-10 | 10000000 |
      | party2 | MXN-0-10 | 10000000 |
      

  Scenario: Maker/taker volume does/does not contribute towards the maker volume fraction respectively (0095-HVMR-013)(0095-HVMR-014)(0095-HVMR-015)(0095-HVMR-016)

    Given the volume rebate program tiers named "vrt":
      | fraction | rebate |
      | 0.0001   | 0.001  |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | party2 | 1    | 50000 | sell           | 0               | 500              |

    # In the following epoch, party1 and party2 are both the maker of a
    # trade but only party1 recevieves a rebate.
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 2      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 50                           |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | 0                            |
    Then the following transfers should happen:
      | from | to     | from account            | to account           | market id | amount | asset   | type                                        |
      |      | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | <market>  | 50     | USD-0-1 | TRANSFER_TYPE_HIGH_MAKER_FEE_REBATE_RECEIVE |

  Examples:
      | market           |
      | BTC/USD-0-1      |
      | MXN-0-10/USD-0-1 |


  Scenario Outline: Trades on auction uncrossing do not contribute towards the maker volume fraction (0095-HVMR-017)(0095-HVMR-018)

    Given the volume rebate program tiers named "vrt":
      | fraction | rebate |
      | 0.0001   | 0.001  |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | error | reference          |
      | aux1  | <market>  | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |       | auction-order-aux1 |
      | aux2  | <market>  | sell | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |       | auction-order-aux2 |
    And the parties cancel the following orders:
      | party | reference          |
      | aux1  | auction-order-aux1 |
      | aux2  | auction-order-aux2 |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market <market string>

    # Exit the PM auction - volume should not contribute towards maker
    # volume fraction of either party1 or party2
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | sell | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market <market string>
    And the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer maker fee | seller maker fee |
      | party1 | party2 | 1    | 50000 |                | 0               | 0                |

    # In the following epoch, party1 and party2 are both the maker of a
    # trade but neither receive a rebate.
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | <market>  | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | <market>  | sell | 2      | 50000 | 2                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | 0                            |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | 0                            |

  Examples:
      | market           | market string      |
      | BTC/USD-0-1      | "BTC/USD-0-1"      |
      | MXN-0-10/USD-0-1 | "MXN-0-10/USD-0-1" |


  Scenario Outline: Volume made in previous window correctly contributes towards maker volume fraction (0095-HVMR-019)(0095-HVMR-020)(0095-HVMR-021)(0095-HVMR-022)

    Given the volume rebate program tiers named "vrt":
      | fraction | rebate |
      | 0.0001   | 0.001  |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length   |
      | id | vrt   | 0                 | <window length> |
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
      | party1 | aux1   | 1    | 50000 | sell           | 0                           | <party1 rebate>              |
      | party2 | aux1   | 1    | 50000 | sell           | 0                           | <party2 rebate>              |

  Examples:
      | market           | window length | epochs between trades | party1 rebate | party2 rebate |
      | BTC/USD-0-1      | 2             | "0"                   | 50            | 50            |
      | BTC/USD-0-1      | 2             | "1"                   | 50            | 50            |
      | BTC/USD-0-1      | 2             | "2"                   | 0             | 50            |
      | MXN-0-10/USD-0-1 | 2             | "0"                   | 50            | 50            |
      | MXN-0-10/USD-0-1 | 2             | "1"                   | 50            | 50            |
      | MXN-0-10/USD-0-1 | 2             | "2"                   | 0             | 50            |


  Scenario Outline: Derivative and spot markets using assets with different quantums scale maker volume correctly (0095-HVMR-023)(0095-HVMR-024)(0095-HVMR-025)

    Given the volume rebate program tiers named "vrt":
      | fraction | rebate |
      | 0.5      | 0.001  |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount  |
      | party1 | USD-0-1  | 1000000 |
      | party2 | MXN-0-10 | 1000000 |
    And the parties place the following orders:
      | party  | market id    | side | volume | price       | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1  | buy  | 1      | <usd price> | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1  | sell | 1      | <usd price> | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/MXN-0-10 | buy  | 1      | <mxn price> | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/MXN-0-10 | sell | 1      | <mxn price> | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price       | aggressor side |
      | party1 | aux1   | 1    | <usd price> | sell           |
      | party2 | aux1   | 1    | <mxn price> | sell           |

    # In the following epoch, party1 and party2 are both the maker of a
    # trade but only party1 recevieves a rebate.
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id    | side | volume | price  | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1  | buy  | 1      | 50000  | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1  | sell | 1      | 50000  | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/MXN-0-10 | buy  | 1      | 500000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/MXN-0-10 | sell | 1      | 500000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    Then debug trades
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price  | aggressor side | buyer high volume maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000  | sell           | 0                           | <party1 rebate>              |
      | party2 | aux1   | 1    | 500000 | sell           | 0                           | <party2 rebate>              |

  Examples:
      | usd market       | mxn market       | usd price | mxn price | party1 rebate | party2 rebate |
      | BTC/USDT-0-1     | BTC/MXN-0-10     | 50001     | 500000    | 50            | 0             |
      | BTC/USDT-0-1     | USD-0-1/MXN-0-10 | 50000     | 500000    | 50            | 500           |
      | MXN-0-1/USD-0-10 | USD-0-1/MXN-0-10 | 50000     | 500001    | 0             | 500           |


  Scenario: A party's rebate is fixed for the epoch and only increases on the next epoch (0095-HVMR-026)(0095-HVMR-027)(0095-HVMR-028)

    Given the volume rebate program tiers named "vrt":
      | fraction       | rebate |
      | 0.000000000001 | 0.001  |
      | 0.500000000001 | 0.010  |
    And the volume rebate program:
      | id | tiers | closing timestamp | window length |
      | id | vrt   | 0                 | 1             |
    And the network moves ahead "1" epochs

    # Place trades such that both parties have exactly half of the maker
    # volume and therefore receive the lower rebate factor.
    Given the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | seller maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 500              |
      | party2 | aux1   | 1    | 50000 | sell           | 500              |

    # Place trades such that party2 has slightly more than half of the
    # maker volume andt therefore will evently receive the higher rebate factor.
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 1      | 50100 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50100 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | seller maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 500              | 50                           |
      | party2 | aux1   | 1    | 50100 | sell           | 501              | 50                           |

    # Place more orders. The rebates are fixed for the epoch so party2
    # should not yet receive the higher rebate factor.
    Given the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | seller maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 500              | 50                           |
      | party2 | aux1   | 1    | 50000 | sell           | 500              | 50                           |

    # In the next epoch the rebate factors are updated and party2 now
    # receives a larger high volume maker fee.
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | error |
      | party1 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
      | party2 | BTC/USD-0-1 | buy  | 1      | 50000 | 0                | TYPE_LIMIT | TIF_GTC |       |
      | aux1   | BTC/USD-0-1 | sell | 1      | 50000 | 1                | TYPE_LIMIT | TIF_GTC |       |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | seller | size | price | aggressor side | seller maker fee | seller high volume maker fee |
      | party1 | aux1   | 1    | 50000 | sell           | 500              | 50                           |
      | party2 | aux1   | 1    | 50000 | sell           | 500              | 500                          |


