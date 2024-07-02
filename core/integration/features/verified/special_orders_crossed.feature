
Feature: Issue: re submit special order would cross

  Background:

    Given the average block duration is "1"
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
    And the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 6     |

    Given the markets:
      | id        | quote name | asset | liquidity monitoring | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params    | decimal places | tick size |
      | ETH/DEC21 | ETH        | ETH   | default-parameters   | default-simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.5                    | 0                         | default-basic | 2              | 10        |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount        |
      | aux1   | ETH   | 1000000000000 |
      | aux2   | ETH   | 1000000000000 |
      | party1 | ETH   | 1000000000000 |
      | party2 | ETH   | 1000000000000 |

  Scenario:

    Given the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference | error |
      | party1 | ETH/DEC21 | 10        | 5                    | buy  | MID              | 10     | 10     | peg-buy-1 |       |
      | party2 | ETH/DEC21 | 10        | 5                    | sell | MID              | 10     | 10     | peg-buy-2 |       |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
      | aux2  | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
    When the opening auction period ends for market "ETH/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"