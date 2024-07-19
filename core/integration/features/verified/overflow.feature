Feature: FCAP liquidations

    vega-market-sim fuzz testing shows parties being liquidated on
    fully collateralised capped future (FCAP) markets.

    For the following test cases, the margin requirements are checked
    and the price moved against the party to check for liquidations.

    - cross margin:
        - party opens short position
        - party reduces short position 
        - party increases short position
        - party opens long position
        - party reduces long position 
        - party increases long position
        - party switches from long to short position
        - party switches from short to long position

  Background:

    # Initialise the network and register the assets
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0     |
      | market.fee.factors.infrastructureFee    | 0     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    And the following assets are registered:
      | id       | decimal places | quantum             |
      | USD-1-10 | 18             | 1000000000000000000 |

    # Initialise the parties and deposit assets
    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount                                                   |
      | lp    | USD-1-10 | 1000000000000000000000000000                             |
      | aux1  | USD-1-10 | 10000000000000000000000000000000000000000000000000000000 |
      | aux2  | USD-1-10 | 10000000000000000000000000000000000000000000000000000000 |

    # Setup the FCAP market in continuous trading
    Given the markets:
      | id            | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places | max price cap | binary | fully collateralised |
      | FCAP/USD-1-10 | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 4              | 1                       | 10000         | true   | true                 |
    And the parties submit the following liquidity provision:
      | id  | party | market id     | commitment amount      | fee | lp type    |
      | lp1 | lp    | FCAP/USD-1-10 | 1000000000000000000000 | 0   | submission |
    And the parties place the following orders:
      | party | market id     | side | volume | price | resulting trades | type       | tif     |
      | aux1  | FCAP/USD-1-10 | buy  | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | FCAP/USD-1-10 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "FCAP/USD-1-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "FCAP/USD-1-10"

  Scenario Outline: Simple test case, party opens long position, margin correctly taken and party never closed out.

    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount  |
      | trader | USD-1-10 | <funds> |

    And the parties place the following orders:
      | party  | market id     | side | volume | price         | resulting trades | type       | tif     |
      | aux1   | FCAP/USD-1-10 | sell | <size> | <entry price> | 0                | TYPE_LIMIT | TIF_GTC |
      | trader | FCAP/USD-1-10 | buy  | <size> | <entry price> | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset    | market id     | margin         |
      | trader | USD-1-10 | FCAP/USD-1-10 | <entry margin> |

    Then the parties place the following orders:
      | party | market id     | side | volume | price         | resulting trades | type       | tif     |
      | aux1  | FCAP/USD-1-10 | buy  | 1      | <final price> | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | FCAP/USD-1-10 | sell | 1      | <final price> | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset    | market id     | margin         |
      | trader | USD-1-10 | FCAP/USD-1-10 | <final margin> |

  Examples:
      | funds | size | entry price | entry margin | final price | final margin |
    #   | 25000000000000000000         | 1            | 2500        | 25000000000000000            | 1           | 10000000000000            |
    #   | 250000000000000000000        | 10           | 2500        | 250000000000000000           | 1           | 100000000000000           |
    #   | 75000000000000000000         | 1            | 7500        | 75000000000000000            | 1           | 10000000000000            |
    #   | 750000000000000000000        | 10           | 7500        | 750000000000000000           | 1           | 100000000000000           |
      | 50966286350000000000000 | 1075465 | 4739 | 50966286350000000000000 | 1 | 10754650000000000000 |