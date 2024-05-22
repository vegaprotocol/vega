
Feature: Pegged orders do not cross

  Aiming for full coverage of edge-cases, check the following:

  - Market decimals > asset decimals
  - Market decimals < asset decimals

  - For each of the above
  - tick size cannot be expressed in asset decimals
  - tick size can just be expressed in asset decimals
  - tick size can be expressed in asset decimals

  - For each of the above
  - offset cannot be expressed in asset decimals
  - offset can just be expressed in asset decimals
  - offset can be expressed in asset decimals

  Background:

    Given the average block duration is "1"
    And the following assets are registered:
      | id       | decimal places | quantum |
      | ETH.1.10 | 1              | 1       |

    And the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 6     |

    And the parties deposit on asset's general account the following amount:
      | party  | asset    | amount        |
      | aux1   | ETH.1.10 | 1000000000000 |
      | aux2   | ETH.1.10 | 1000000000000 |
      | party1 | ETH.1.10 | 1000000000000 |
      | party2 | ETH.1.10 | 1000000000000 |

  Scenario Outline: # Market decimals > asset decimals

    Given the markets:
      | id             | quote name | asset    | liquidity monitoring | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params    | decimal places | tick size   |
      | ETH.1.10/DEC21 | ETH.1.10   | ETH.1.10 | default-parameters   | default-simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.5                    | 0                         | default-basic | 2              | <tick size> |
    Given the parties place the following pegged orders:
      | party  | market id      | side | pegged reference | volume | offset   | reference | error   |
      | party1 | ETH.1.10/DEC21 | buy  | MID              | 10     | <offset> | peg-buy   | <error> |
      | party2 | ETH.1.10/DEC21 | sell | MID              | 10     | <offset> | peg-sell  | <error> |
    And the parties place the following orders:
      | party | market id      | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH.1.10/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux1  | ETH.1.10/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux2  | ETH.1.10/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
      | aux2  | ETH.1.10/DEC21 | sell | 1      | <bo>  | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
    When the opening auction period ends for market "ETH.1.10/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH.1.10/DEC21"

    Examples:
      | bo   | tick size | offset | error                                  |
      | 1010 | 1         | 1      | invalid offset - pegged mid will cross |
      | 1010 | 1         | 10     |                                        |
      | 1010 | 1         | 100    |                                        |
      | 1010 | 10        | 1      | OrderError: price not in tick size     |
      | 1010 | 10        | 10     |                                        |
      | 1010 | 10        | 100    |                                        |
      | 1100 | 100       | 1      | OrderError: price not in tick size     |
      | 1100 | 100       | 10     | OrderError: price not in tick size     |
      | 1100 | 100       | 100    |                                        |


  Scenario Outline: # Market decimals < asset decimals

    Given the markets:
      | id             | quote name | asset    | liquidity monitoring | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params    | decimal places | tick size   |
      | ETH.1.10/DEC21 | ETH.1.10   | ETH.1.10 | default-parameters   | default-simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.5                    | 0                         | default-basic | 0              | <tick size> |
    Given the parties place the following pegged orders:
      | party  | market id      | side | pegged reference | volume | offset   | reference | error   |
      | party1 | ETH.1.10/DEC21 | buy  | MID              | 10     | <offset> | peg-buy   | <error> |
      | party2 | ETH.1.10/DEC21 | sell | MID              | 10     | <offset> | peg-sell  | <error> |
    And the parties place the following orders:
      | party | market id      | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH.1.10/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux1  | ETH.1.10/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux2  | ETH.1.10/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
      | aux2  | ETH.1.10/DEC21 | sell | 1      | <bo>  | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
    When the opening auction period ends for market "ETH.1.10/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH.1.10/DEC21"

    Examples:
      | bo   | tick size | offset | error                              |
      | 1001 | 1         | 1      |                                    |
      | 1001 | 1         | 10     |                                    |
      | 1001 | 1         | 100    |                                    |
      | 1010 | 10        | 1      | OrderError: price not in tick size |
      | 1010 | 10        | 10     |                                    |
      | 1010 | 10        | 100    |                                    |
      | 1100 | 100       | 1      | OrderError: price not in tick size |
      | 1100 | 100       | 10     | OrderError: price not in tick size |
      | 1100 | 100       | 100    |                                    |
