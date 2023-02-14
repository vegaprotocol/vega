Feature: Set up a market, with an opening auction, then uncross the book in presence of wash trades

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |

  Scenario: Set up opening auction with wash trades and uncross
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 50000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 50000             | 0.1 | sell | MID              | 50         | 100    | submission |
    # place orders and generate trades
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party1 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | party2 | ETH/DEC19 | sell | 3      | 12000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    Then the opening auction period ends for market "ETH/DEC19"

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 10000      | TRADING_MODE_CONTINUOUS | 50000        | 50000          | 5             |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 5    | party1 |
      | party1 | 10000 | 5    | party2 |

    Then the orders should have the following status:
      | party  | reference | status           |
      | party3 | t3-b-1    | STATUS_ACTIVE    |
      | party4 | t4-s-1    | STATUS_ACTIVE    |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party1 | t1-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_FILLED    |
      | party2 | t2-s-2    | STATUS_FILLED    |
      | party2 | t2-s-3    | STATUS_CANCELLED |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 5      | 0              | 0            |
      | party2 | -5     | 0              | 0            |
