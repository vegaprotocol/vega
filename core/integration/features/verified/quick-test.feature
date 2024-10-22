Feature: Quick-test

  Scenario:
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0     |
      | market.fee.factors.infrastructureFee    | 0     |
      | network.markPriceUpdateMaximumFrequency | 10s   |
      | validators.epoch.length                 | 1000s |

    # Initialise the markets
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD-1-10 | 0              | 1       |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.01 | 0.01  | 100         | -100          | 0.2                    |
    And the margin calculator named "margin-calculator":
      | search factor | initial factor | release factor |
      | 1             | 1              | 1              |
    And the markets:
      | id           | quote name | asset    | risk model        | margin calculator | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD-1-10 | ETH        | USD-1-10 | simple-risk-model | margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset    | amount        |
      | aux1                                                             | USD-1-10 | 1000000000000 |
      | aux2                                                             | USD-1-10 | 1000000000000 |
      | party1                                                           | USD-1-10 | 1000000000000 |
      | party2                                                           | USD-1-10 | 1000000000000 |
      | party3                                                           | USD-1-10 | 1000000000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 1000000000000 |




    # Exit opening auctions
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD-1-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "11" blocks
    Then the parties should have the following account balances:
      | party | asset    | market id    | margin | general       |
      | aux1  | USD-1-10 | ETH/USD-1-10 | 0      | 1000000000000 |
      | aux2  | USD-1-10 | ETH/USD-1-10 | 0      | 1000000000000 |
    




    Given the parties place the following orders:
      | party  | market id    | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USD-1-10 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset    | market id    | margin | general      |
      | party1 | USD-1-10 | ETH/USD-1-10 | 10     | 999999999990 |
      | party2 | USD-1-10 | ETH/USD-1-10 | 10     | 999999999990 |


    # # X
    Given the parties place the following orders:
      | party  | market id    | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/USD-1-10 | sell | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/USD-1-10 | buy  | 1      | 990   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset    | market id    | margin | general       |
      | party1 | USD-1-10 | ETH/USD-1-10 | 0      | 1000000000000 |
      | party2 | USD-1-10 | ETH/USD-1-10 | 10     | 999999999990  |
      | party3 | USD-1-10 | ETH/USD-1-10 | 10     | 999999999990  |

    When the parties withdraw the following assets:
      | party  | asset    | amount        | error |
      | party1 | USD-1-10 | 1000000000000 |       |


    Then the network moves ahead "100" blocks
    Then debug transfers
    Then the parties should have the following account balances:
      | party  | asset    | market id    | margin | general      |
      | party1 | USD-1-10 | ETH/USD-1-10 | 0      | 0            |
      | party2 | USD-1-10 | ETH/USD-1-10 | 11     | 999999999989 |
      | party3 | USD-1-10 | ETH/USD-1-10 | 11     | 999999999989 |
