Feature: Over leveraged trader can pay fees with released margin

    Ticket #2010 (https://github.com/vegaprotocol/specs/issues/2010)
    states a party should be able to cover any expected fees when
    reducing their position with a zero balance in their general
    account.

    Test checks this is true for the following combinations:
    - long and shorts positions
    - market and limit orders
    - exit price above and below the mark price at exit


  Background:

    # Initialise the network
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.00  |
      | market.fee.factors.infrastructureFee    | 0.01  |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    # Initialise the market
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD-1-10 | 0              | 1       |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.1                    |
    And the markets:
      | id           | quote name | asset    | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD-1-10 | ETH        | USD-1-10 | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0                      | 0                         | default-futures | 0              | 0                       |

    # Initialise the parties
    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount  |
      | aux1   | USD-1-10 | 1000000 |
      | aux2   | USD-1-10 | 1000000 |
      | trader | USD-1-10 | 1250    |

    # Exit opening auctions
    When the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/USD-1-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"


  Scenario Outline: With no funds in the general account, reducing long position with a limit order at various prices

    # Open a position
    Given the parties place the following orders:
      | party  | market id    | side | volume | price | resulting trades | type       | tif     |
      | trader | ETH/USD-1-10 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/USD-1-10 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | trader | 10     | 0              | 0            |
    And the parties should have the following account balances:
      | party  | asset    | market id    | margin | general |
      | trader | USD-1-10 | ETH/USD-1-10 | 1200   | 50      |
    And the parties should have the following margin levels:
      | party  | market id    | maintenance | search | initial | release |
      | trader | ETH/USD-1-10 | 1000        | 1100   | 1200    | 1400    |

    # Empty general account
    Given the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 975   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 975   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party  | asset    | market id    | margin | general |
      | trader | USD-1-10 | ETH/USD-1-10 | 1000   | 0       |
    And the parties should have the following margin levels:
      | party  | market id    | maintenance | search | initial | release |
      | trader | ETH/USD-1-10 | 975         | 1072   | 1170    | 1365    |

    # Attempt to reduce position
    Given the parties place the following orders:
      | party  | market id    | side | volume | price   | resulting trades | type       | tif     |
      | aux2   | ETH/USD-1-10 | buy  | <size> | <price> | 0                | TYPE_LIMIT | TIF_GTC |
      | trader | ETH/USD-1-10 | sell | <size> | <price> | 1                | <type>     | TIF_IOC |
    When the network moves ahead "1" blocks
    And the following trades should be executed:
      | buyer | price   | size   | seller | seller fee |
      | aux2  | <price> | <size> | trader | <fee>      |
    And the parties should have the following account balances:
      | party  | asset    | market id    | margin   | general   |
      | trader | USD-1-10 | ETH/USD-1-10 | <margin> | <general> |

  Examples:
    # Table contains inputs for the trade reducing the over leveraged
    # position and the resulting account balances.
      | size | price | fee | margin | general | type        |
      | 1    | 950   | 10  | 0      | 0       | TYPE_LIMIT  |
      | 9    | 950   | 86  | 114    | 550     | TYPE_LIMIT  |
      | 10   | 970   | 97  | 0      | 853     | TYPE_LIMIT  |
      | 1    | 975   | 10  | 990    | 0       | TYPE_LIMIT  |
      | 9    | 975   | 88  | 117    | 795     | TYPE_LIMIT  |
      | 10   | 975   | 98  | 0      | 902     | TYPE_LIMIT  |
      | 1    | 1000  | 10  | 1240   | 0       | TYPE_LIMIT  |
      | 9    | 1000  | 90  | 120    | 1040    | TYPE_LIMIT  |
      | 10   | 1000  | 100 | 0      | 1150    | TYPE_LIMIT  |
      | 1    | 950   | 10  | 0      | 0       | TYPE_MARKET |
      | 9    | 950   | 86  | 114    | 550     | TYPE_MARKET |
      | 10   | 970   | 97  | 0      | 853     | TYPE_MARKET |
      | 1    | 975   | 10  | 990    | 0       | TYPE_MARKET |
      | 9    | 975   | 88  | 117    | 795     | TYPE_MARKET |
      | 10   | 975   | 98  | 0      | 902     | TYPE_MARKET |
      | 1    | 1000  | 10  | 1240   | 0       | TYPE_MARKET |
      | 9    | 1000  | 90  | 120    | 1040    | TYPE_MARKET |
      | 10   | 1000  | 100 | 0      | 1150    | TYPE_MARKET |


  Scenario Outline: With no funds in the general account, reducing short position with a limit order at various prices

    # Open a position
    Given the parties place the following orders:
      | party  | market id    | side | volume | price | resulting trades | type       | tif     |
      | trader | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | trader | -10    | 0              | 0            |
    And the parties should have the following account balances:
      | party  | asset    | market id    | margin | general |e
      | trader | USD-1-10 | ETH/USD-1-10 | 1200   | 50      |
    And the parties should have the following margin levels:
      | party  | market id    | maintenance | search | initial | release |
      | trader | ETH/USD-1-10 | 1000        | 1100   | 1200    | 1400    |

    # Empty general account
    Given the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1020  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party  | asset    | market id    | margin | general |
      | trader | USD-1-10 | ETH/USD-1-10 | 1050   | 0       |
    And the parties should have the following margin levels:
      | party  | market id    | maintenance | search | initial | release |
      | trader | ETH/USD-1-10 | 1020        | 1122   | 1224    | 1428    |

    # Attempt to reduce position
    Given the parties place the following orders:
      | party  | market id    | side | volume | price   | resulting trades | type       | tif     |
      | aux2   | ETH/USD-1-10 | sell | <size> | <price> | 0                | TYPE_LIMIT | TIF_GTC |
      | trader | ETH/USD-1-10 | buy  | <size> | <price> | 1                | <type>     | TIF_IOC |
    When the network moves ahead "1" blocks
    Then debug trades
    And the following trades should be executed:
      | buyer  | price   | size   | seller | buyer fee |
      | trader | <price> | <size> | aux2   | <fee>     |
    And the parties should have the following account balances:
      | party  | asset    | market id    | margin   | general   |
      | trader | USD-1-10 | ETH/USD-1-10 | <margin> | <general> |

  Examples:
    # Table contains inputs for the trade reducing the over leveraged
    # position and the resulting account balances.
      | size | price | fee | margin | general | type        |
      | 1    | 1040  | 11  | 0      | 0       | TYPE_LIMIT  |
      | 9    | 1040  | 94  | 124    | 632     | TYPE_LIMIT  |
      | 10   | 1040  | 104 | 0      | 746     | TYPE_LIMIT  |
      | 1    | 1020  | 11  | 1039   | 0       | TYPE_LIMIT  |
      | 9    | 1020  | 92  | 122    | 836     | TYPE_LIMIT  |
      | 10   | 1020  | 102 | 0      | 948     | TYPE_LIMIT  |
      | 1    | 1000  | 10  | 1240   | 0       | TYPE_LIMIT  |
      | 9    | 1000  | 90  | 120    | 1040    | TYPE_LIMIT  |
      | 10   | 1000  | 100 | 0      | 1150    | TYPE_LIMIT  |
      | 1    | 1040  | 11  | 0      | 0       | TYPE_MARKET |
      | 9    | 1040  | 94  | 124    | 632     | TYPE_MARKET |
      | 10   | 1040  | 104 | 0      | 746     | TYPE_MARKET |
      | 1    | 1020  | 11  | 1039   | 0       | TYPE_MARKET |
      | 9    | 1020  | 92  | 122    | 836     | TYPE_MARKET |
      | 10   | 1020  | 102 | 0      | 948     | TYPE_MARKET |
      | 1    | 1000  | 10  | 1240   | 0       | TYPE_MARKET |
      | 9    | 1000  | 90  | 120    | 1040    | TYPE_MARKET |
      | 10   | 1000  | 100 | 0      | 1150    | TYPE_MARKET |