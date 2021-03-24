Feature: Set up a market, with an opening auction, then uncross the book

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: set up 2 traders with balance
    # setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount    |
      | trader1 | BTC   | 100000000 |
      | trader2 | BTC   | 100000000 |
      | trader3 | BTC   | 100000000 |
      | trader4 | BTC   | 100000000 |

    # place orders and generate trades
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | trader4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | trader1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    Then the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 25201       | 27721  | 30241   | 65521   |
      | trader2 | ETH/DEC19 | 23899       | 26289  | 28679   | 57458   |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general  |
      | trader1 | BTC   | ETH/DEC19 | 30241  | 99969759 |
      | trader2 | BTC   | ETH/DEC19 | 28679  | 99971321 |
    And "trader1" withdraws "99969759" from the "BTC" account
    And "trader2" withdraws "99971321" from the "BTC" account
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | BTC   | ETH/DEC19 | 30241  | 0       |
      | trader2 | BTC   | ETH/DEC19 | 28679  | 0       |
    Then the opening auction period for market "ETH/DEC19" ends
    ## We're seeing these events twice for some reason
    And executed trades:
      | buyer   | price | size | seller  |
      | trader1 | 10000 | 3    | trader2 |
      | trader1 | 10000 | 2    | trader2 |
      | trader1 | 10000 | 3    | trader2 |
    And the mark price for the market "ETH/DEC19" is "10000"
    ## Network for distressed trader1 -> cancelled, nothing on the book is remaining
    Then verify the status of the order reference:
      | trader  | reference | status           |
      | trader1 | t1-b-1    | STATUS_FILLED    |
      | trader2 | t2-s-1    | STATUS_FILLED    |
      | trader1 | t1-b-2    | STATUS_CANCELLED |
      | trader2 | t2-s-2    | STATUS_CANCELLED |
      | trader1 | t1-b-3    | STATUS_CANCELLED |
      | trader2 | t2-s-3    | STATUS_FILLED    |
    And the following transfers happened:
      | from    | to      | from account   | to account      | market id | amount | asset |
      | trader2 | trader2 | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_GENERAL | ETH/DEC19 | 9479   | BTC   |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader2 | BTC   | ETH/DEC19 | 19200  | 9479    |
      | trader1 | BTC   | ETH/DEC19 | 30241  | 0       |
