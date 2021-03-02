Feature: Set up a market, with an opening auction, then uncross the book


  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r  | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | ETH      | BTC       | BTC   | 100       | simple     | 0.1       | 0.1       | -1 | -1 | -1    | 1.4            | 1.2            | 1.1           | 100             | 1           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |

  Scenario: set up 2 traders with balance
    # setup accounts
    Given the following traders:
      | name    | amount    |
      | trader1 | 100000000 |
      | trader2 | 100000000 |
      | trader3 | 100000000 |
      | trader4 | 100000000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | BTC   |
      | trader2 | BTC   |
      | trader3 | BTC   |
      | trader4 | BTC   |

    # place orders and generate trades
    Then traders place following orders with references:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000   | 0                | TYPE_LIMIT  | TIF_GTC | t3-b-1    |
      | trader4 | ETH/DEC19 | sell | 1      | 11000  | 0                | TYPE_LIMIT  | TIF_GTC | t4-s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000  | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 5      | 10000  | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000  | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 10001  | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-2    |
      | trader1 | ETH/DEC19 | buy  | 4      | 3000   | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 3      | 3000   | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-3    |
#   And dump orders
    Then the margins levels for the traders are:
      | trader  | id        | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |       25201 |  27721 |   30241 |   65521 |
      # | trader2 | ETH/DEC19 |       23899 |  26289 |   28679 |   62137 |
      | trader2 | ETH/DEC19 |       23899 |  26289 |   28679 |   57458 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general  |
#     | trader1 | BTC   | ETH/DEC19 |  11200 | 99988800 |
#     | trader2 | BTC   | ETH/DEC19 |  10899 | 99989101 |
      | trader1 | BTC   | ETH/DEC19 |  30241 | 99969759 |
      | trader2 | BTC   | ETH/DEC19 |  28679 | 99971321 |
    And traders withdraw balance:
      | trader  | asset | amount   |
      | trader1 | BTC   | 99969759 |
      | trader2 | BTC   | 99971321 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general  |
      | trader1 | BTC   | ETH/DEC19 |  30241 | 0        |
      | trader2 | BTC   | ETH/DEC19 |  28679 | 0        |
#   And dump transfers
    Then the opening auction period for market "ETH/DEC19" ends
    ## We're seeing these events twice for some reason
    And executed trades:
      | buyer   | price | size | seller  |
      | trader1 | 10000 | 3    | trader2 |
      | trader1 | 10000 | 2    | trader2 |
      | trader1 | 10000 | 3    | trader2 |
    And the mark price for the market "ETH/DEC19" is "10000"
#   And dump trades
#   And dump transfers
    ## Network for distressed trader1 -> cancelled, nothing on the book is remaining
    Then verify the status of the order reference:
      | trader  | reference | status           |
      | trader1 | t1-b-1    | STATUS_FILLED    |
      | trader2 | t2-s-1    | STATUS_FILLED    |
      | trader1 | t1-b-2    | STATUS_CANCELLED |
      | trader2 | t2-s-2    | STATUS_CANCELLED |
      | trader1 | t1-b-3    | STATUS_CANCELLED |
      | trader2 | t2-s-3    | STATUS_FILLED    |
#   And dump trades
#   And dump transfers
    And the following transfers happened:
      | from    | to      | from account type   | to account type      | market ID | amount | asset |
      | trader2 | trader2 | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_GENERAL | ETH/DEC19 | 9479   | BTC   |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general  |
      | trader2 | BTC   | ETH/DEC19 | 19200  | 9479     |
      | trader1 | BTC   | ETH/DEC19 | 30241  | 0        |
