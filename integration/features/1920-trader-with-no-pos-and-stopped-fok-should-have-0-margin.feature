Feature: test for issue 1920

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |

  Scenario: a trader place a new order in the system, margin are calculated, then the order is stopped, the margin is released
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount    |
      | trader1 | ETH   | 10000     |
      | trader2 | ETH   | 100000000 |
      | trader3 | ETH   | 100000000 |
      | trader4 | ETH   | 100000000 |
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader3 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |
      | trader4 | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | trader4 | ETH/DEC19 | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |

    Then the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 10000   |
