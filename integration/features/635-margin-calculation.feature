Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  @ignore
  Scenario: Traded out position but monies left in margin account
    # setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount     |
      | traderGuy        | BTC   | 1000000000 |
      | sellSideProvider | BTC   | 1000000000 |
      | buySideProvider  | BTC   | 1000000000 |
      | aux              | BTC   | 1000000000 |
      | aux2             | BTC   | 1000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price      | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 8700000    | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 25000000   | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 10300000   | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10300000   | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period for market "ETH/DEC19" ends
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"


    # setup previous mark price
    Then traders place the following orders:
      | trader           | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # setup orderbook
    When traders place the following orders:
      | trader           | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | sellSideProvider | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideProvider | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | buySideProvider  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | buySideProvider  | ETH/DEC19 | buy  | 15     | 9000000  | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
# buy 13@150
    Then traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And debug trades
# checking margins
    Then traders have the following account balances:
      | trader    | asset | market id | margin    | general   |
      | traderGuy | BTC   | ETH/DEC19 | 394400032 | 611199968 |
# checking margins levels
    Then the margins levels for the traders are:
      | trader    | market id | maintenance | search    | initial   | release   |
      | traderGuy | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
