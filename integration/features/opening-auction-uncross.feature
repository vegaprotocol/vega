Feature: Set up a market, with an opening auction, then uncross the book

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |

  Scenario: set up 2 traders with balance
    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount    |
      | trader1 | BTC   | 100000000 |
      | trader2 | BTC   | 100000000 |
      | trader3 | BTC   | 100000000 |
      | trader4 | BTC   | 100000000 |

    # place orders and generate trades
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | trader4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | trader1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    Then the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 25200       | 27720  | 30240   | 65520   |
      | trader2 | ETH/DEC19 | 23900       | 26290  | 28680   | 57460   |
      # values before uint stuff
      #| trader1 | ETH/DEC19 | 25201       | 27721  | 30241   | 65521   |
      #| trader2 | ETH/DEC19 | 23899       | 26289  | 28679   | 57458   |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general  |
      | trader1 | BTC   | ETH/DEC19 | 30240  | 99969760 |
      | trader2 | BTC   | ETH/DEC19 | 28680  | 99971320 |
      # values before uint
      #| trader1 | BTC   | ETH/DEC19 | 30241  | 99969759 |
    When the traders withdraw the following assets:
      | trader  | asset | amount   |
      | trader1 | BTC   | 99969760 |
      | trader2 | BTC   | 99971320 |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | BTC   | ETH/DEC19 | 30240  | 0       |
      | trader2 | BTC   | ETH/DEC19 | 28680  | 0       |
      # values before uint
      #| trader1 | BTC   | ETH/DEC19 | 30241  | 0       |
    Then the opening auction period ends for market "ETH/DEC19"
    ## We're seeing these events twice for some reason
    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 10000 | 3    | trader2 |
      | trader1 | 10000 | 2    | trader2 |
      | trader1 | 10000 | 3    | trader2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    ## Network for distressed trader1 -> cancelled, nothing on the book is remaining
    Then the orders should have the following status:
      | trader  | reference | status           |
      | trader1 | t1-b-1    | STATUS_FILLED    |
      | trader2 | t2-s-1    | STATUS_FILLED    |
      | trader1 | t1-b-2    | STATUS_CANCELLED |
      | trader2 | t2-s-2    | STATUS_CANCELLED |
      | trader1 | t1-b-3    | STATUS_CANCELLED |
      | trader2 | t2-s-3    | STATUS_FILLED    |
    And the following transfers should happen:
      | from    | to      | from account        | to account           | market id | amount | asset |
      | trader2 | trader2 | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_GENERAL | ETH/DEC19 | 9480   | BTC   |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader2 | BTC   | ETH/DEC19 | 19200  | 9480    |
      | trader1 | BTC   | ETH/DEC19 | 30240  | 0       |
      # values before uint
      #| trader1 | BTC   | ETH/DEC19 | 30241  | 0       |
