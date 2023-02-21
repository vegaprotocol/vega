Feature: test for issue 1920

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: a party place a new order in the system, margin are calculated, then the order is stopped, the margin is released
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000     |
      | party2 | ETH   | 100000000 |
      | party3 | ETH   | 100000000 |
      | party4 | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | party3 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |
      | party4 | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | party4 | ETH/DEC19 | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 10000   |
