Feature: Set up a market, with an opening auction, then uncross the book


  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r  | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | ETH      | BTC       | BTC   | 100       | simple     | 0.1       | 0.1       | -1 | -1 | -1    | 1.4            | 1.2            | 1.1           | 100             | 1           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |

  Scenario: set up 2 traders with balance
    # setup accounts
    Given the following traders:
      | name    | amount    |
      | trader1 | 100000000 |
      | trader2 | 100000000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | BTC   |
      | trader2 | BTC   |

    # place orders and generate trades
    Then traders place following orders with references:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000  | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 5      | 10000  | 0                | TYPE_LIMIT  | TIF_GFA | t2_s-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 10000  | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 5      | 10001  | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-2    |
      | trader1 | ETH/DEC19 | buy  | 4      | 3000   | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 3      | 3000   | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-3    |
    And dump orders
    Then the margins levels for the traders are:
      | trader  | id        | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |       11200 |  12320 |   13440 |   27679 |
      | trader2 | ETH/DEC19 |       10899 |  11989 |   13079 |   27258 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general  |
#     | trader1 | BTC   | ETH/DEC19 |  11200 | 99988800 |
#     | trader2 | BTC   | ETH/DEC19 |  10899 | 99989101 |
      | trader1 | BTC   | ETH/DEC19 |  13440 | 99986560 |
      | trader2 | BTC   | ETH/DEC19 |  13079 | 99986921 |
    And traders withdraw balance:
      | trader  | asset | amount   |
      | trader1 | BTC   | 99986560 |
      | trader2 | BTC   | 99986921 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general  |
      | trader1 | BTC   | ETH/DEC19 |  13440 | 0        |
      | trader2 | BTC   | ETH/DEC19 |  13079 | 0        |
    Then the opening auction period for market "ETH/DEC19" ends
    And dump orders
