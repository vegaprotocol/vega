Feature: Cannot place an network order

  Background:
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: an order is rejected if a trader try to place an order with type NETWORK
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | ETH   | 1      |
    When the traders place the following orders:
      | trader  | market id | side | volume | price | type         | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | TYPE_NETWORK | TIF_GTC | ref-1     |
    Then the system should return error "invalid order type"
