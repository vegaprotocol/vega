Feature: Close a filled order twice

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Traders place an order, a trade happens, and orders are cancelled after being filled
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | aux              | BTC   | 100000    |
      | aux2             | BTC   | 100000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And traders place the following orders:
      | trader  | market id        | side | volume | price | resulting trades | type        | tif     | reference |
      | aux     | ETH/DEC19        | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux     | ETH/DEC19        | sell | 1      | 200   | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |
      | aux2    | ETH/DEC19        | buy  | 1      | 120   | 0                | TYPE_LIMIT  | TIF_GTC | ref-3     |
      | aux     | ETH/DEC19        | sell | 1      | 120   | 0                | TYPE_LIMIT  | TIF_GTC | ref-4     |
    Then the opening auction period for market "ETH/DEC19" ends
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    # setup orderbook
    And traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 10     | 120   | 1                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    When traders cancel the following orders:
      | trader          | reference      |
      | buySideProvider | buy-provider-1 |
    Then the system should return error "unable to find the order in the market"
    When traders cancel the following orders:
      | trader          | reference      |
      | buySideProvider | buy-provider-1 |
    Then the system should return error "unable to find the order in the market"
    When traders cancel the following orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
    Then the insurance pool balance is "0" for the market "ETH/DEC19"
    Then Cumulated balance for all accounts is worth "200200000"
