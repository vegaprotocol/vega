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
      | party5 | BTC   | 1         |
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
      | party2 | ETH/DEC19 | 10901       | 11991  | 13081   | 15261   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | BTC   | ETH/DEC19 | 13440  | 99986560 |
      | party2 | BTC   | ETH/DEC19 | 13081  | 99986919 |
    When the parties withdraw the following assets:
      | party  | asset | amount   |
      | party1 | BTC   | 99949760 |
      | party2 | BTC   | 99951320 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 13440  | 36800   |
      | party2 | BTC   | ETH/DEC19 | 13081  | 35599   |
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
      | party2 | BTC   | ETH/DEC19 | 9600   | 39080   |
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
 
  Scenario: Party cannot afford pegged orders upon uncrossing so they get stopped
    Given the following network parameters are set:
      | name                                                | value |
      | limits.markets.maxPeggedOrders                      | 10    | 
    When the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party4 | ETH/DEC19 | buy  | 100000 | BID              | 1      |
      | party4 | ETH/DEC19 | buy  | 100000 | MID              | 1      |
      | party5 | ETH/DEC19 | sell | 100000 | ASK              | 1      |
      | party5 | ETH/DEC19 | sell | 100000 | MID              | 1      |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GFA |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |
    
    Then the pegged orders should have the following states:
      | party  | market id | side | volume | reference | offset | price | status        |
      | party4 | ETH/DEC19 | buy  | 100000 | BID       | 1      | 0     | STATUS_PARKED |
      | party4 | ETH/DEC19 | buy  | 100000 | MID       | 1      | 0     | STATUS_PARKED |
      | party5 | ETH/DEC19 | sell | 100000 | ASK       | 1      | 0     | STATUS_PARKED |
      | party5 | ETH/DEC19 | sell | 100000 | MID       | 1      | 0     | STATUS_PARKED |
    
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |
    And the pegged orders should have the following states:
      | party  | market id | side | volume | reference | offset | price | status           |
      | party4 | ETH/DEC19 | buy  | 100000 | BID       | 1      | 899   | STATUS_ACTIVE    |
      | party4 | ETH/DEC19 | buy  | 100000 | MID       | 1      | 999   | STATUS_ACTIVE    |
      | party5 | ETH/DEC19 | sell | 100000 | ASK       | 1      | 1101  | STATUS_CANCELLED |
      | party5 | ETH/DEC19 | sell | 100000 | MID       | 1      | 1001  | STATUS_CANCELLED |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1101  |      0 |
      | sell | 1100  |     83 |
      | sell | 1001  |      0 |
      | buy  | 999   | 100000 |
      | buy  | 900   |    101 |
      | buy  | 899   | 100000 |
    
    # Move the best bid and assure the orders don't resurrect
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the pegged orders should have the following states:
      | party  | market id | side | volume | reference | offset | price | status           |
      | party4 | ETH/DEC19 | buy  | 100000 | BID       | 1      | 899   | STATUS_ACTIVE    |
      | party4 | ETH/DEC19 | buy  | 100000 | MID       | 1      | 949   | STATUS_ACTIVE    |
      | party5 | ETH/DEC19 | sell | 100000 | ASK       | 1      | 1101  | STATUS_CANCELLED |
      | party5 | ETH/DEC19 | sell | 100000 | MID       | 1      | 1001  | STATUS_CANCELLED |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1101  |      0 |
      | sell | 1100  |      1 |
      | sell | 1050  |     86 |
      | sell | 1001  |      0 |
      | sell | 1000  |      1 |
      | sell | 999   |      0 |
      | sell | 951   |      0 |
      | buy  | 999   |      0 |
      | buy  | 949   | 100000 |
      | buy  | 900   |      1 |
      | buy  | 899   | 100000 |
      | buy  | 850   |    106 |
