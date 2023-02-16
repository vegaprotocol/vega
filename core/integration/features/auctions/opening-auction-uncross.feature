Feature: Set up a market, with an opening auction, then uncross the book

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

  Scenario: set up 2 parties with balance
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | party1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | party2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 11200       | 12320  | 13440   | 15680   |
      | party2 | ETH/DEC19 | 10900       | 11990  | 13080   | 15260   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | BTC   | ETH/DEC19 | 13440  | 99986560 |
      | party2 | BTC   | ETH/DEC19 | 13080  | 99986920 |
    When the parties withdraw the following assets:
      | party  | asset | amount   |
      | party1 | BTC   | 99949760 |
      | party2 | BTC   | 99951320 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 13440  | 36800   |
      | party2 | BTC   | ETH/DEC19 | 13080  | 35600   |
    Then the opening auction period ends for market "ETH/DEC19"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 3    | party2 |
      | party1 | 10000 | 2    | party2 |
      | party1 | 10000 | 3    | party2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    When the network moves ahead "1" blocks
    Then the orders should have the following status:
      | party  | reference | status           |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party2 | t2-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_CANCELLED |
      | party2 | t2-s-2    | STATUS_CANCELLED |
      | party1 | t1-b-3    | STATUS_CANCELLED |
      | party2 | t2-s-3    | STATUS_FILLED    |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party2 | BTC   | ETH/DEC19 | 47040  | 1640    |
      | party1 | BTC   | ETH/DEC19 | 48960  | 1280    |

  Scenario: Uncross auction via order amendment
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 9999  | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |

    Then the network moves ahead "2" blocks
    And the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party1 | t1-b-1    | 10000 | 2          | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |
    And the mark price should be "10000" for the market "ETH/DEC19"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 5    | party2 |
