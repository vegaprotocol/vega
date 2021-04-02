Feature: test for issue 1920

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader place a new order in the system, margin are calculated, then the order is stopped, the margin is released
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount    |
      | trader1 | ETH   | 10000     |
      | trader2 | ETH   | 100000000 |
      | trader3 | ETH   | 100000000 |
      | trader4 | ETH   | 100000000 |
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader3 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |
      | trader4 | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | trader4 | ETH/DEC19 | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "1000"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |

    Then the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 10000   |
