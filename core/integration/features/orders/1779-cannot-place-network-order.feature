Feature: Cannot place an network order

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future |

  Scenario: an order is rejected if a party try to place an order with type NETWORK (0014-ORDT-005)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 1      |
    When the parties place the following orders:
      | party  | market id | side | volume | price | type         | tif     | reference | error              |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | TYPE_NETWORK | TIF_GTC | ref-1     | invalid order type |
