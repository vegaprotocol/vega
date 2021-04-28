Feature: Test LP orders

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 1      |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: create liquidity provisions
    Given the traders deposit on asset's general account the following amount:
      | trader           | asset | amount    |
      | trader1          | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    When the traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | auxiliary        | ETH/DEC19 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1          |
      | auxiliary        | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1          |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2          |
      | auxiliary        | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2          |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | trader1          | ETH/DEC19 | buy  | 500    | 110   | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-1        |
      | trader1          | ETH/DEC19 | sell | 500    | 120   | 0                | TYPE_LIMIT | TIF_GTC | lp-ref-2        |
    Then the orders should have the following states:
      | trader           | market id | side | volume | price | status        |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | STATUS_ACTIVE |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | STATUS_ACTIVE |
    And clear order events
    Then the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader1 | ETH/DEC19 | 10000             | 0.1 | buy        | BID             | 500              | -10          |
      | lp1 | trader1 | ETH/DEC19 | 10000             | 0.1 | sell       | ASK             | 500              | 10           |
    Then I see the LP events:
      | id  | party   | market    | commitment amount | status        |
      | lp1 | trader1 | ETH/DEC19 | 10000             | STATUS_ACTIVE |
    Then the orders should have the following states:
      | trader  | market id | side | volume | price | status        |
      | trader1 | ETH/DEC19 | buy  | 450    | 100   | STATUS_ACTIVE |
      | trader1 | ETH/DEC19 | sell | 308    | 130   | STATUS_ACTIVE |
