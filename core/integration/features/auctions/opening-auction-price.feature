Feature: Set up a market, create indiciative price different to actual opening auction uncross price

  Background:
    Given the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 2           | -3            | 0.2                    |
    Given the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | my-simple-risk-model | default-margin-calculator | 5                | default-none | default-basic    | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 5     |
      | network.floatingPointUpdates.delay      | 10s   |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @OpenIP
  Scenario: Simple test with different indicative price before auction uncross
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | party5 | BTC   | 100000000 |
      | party6 | BTC   | 100000000 |
      | party7 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # Start market with some dead time
    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    # Ensure an indicative price/volume of 10, although we will not uncross at this price point
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party6 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GFA | t6-b-1    |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC19 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GFA | t5-s-1    |
    # place orders to set the actual price point at which we'll uncross to be 10000
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | party1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | party2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
      | party7 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GFA | t7-s-1    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 11200       | 12320  | 13440   | 15680   |
      | party2 | ETH/DEC19 | 10901       | 11991  | 13081   | 15261   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | BTC   | ETH/DEC19 | 13440  | 99986560 |
      | party2 | BTC   | ETH/DEC19 | 13081  | 99986919 |
    When the opening auction period ends for market "ETH/DEC19"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 1    | party5 |
      | party1 | 10000 | 3    | party2 |
      | party1 | 10000 | 1    | party2 |
      | party1 | 10000 | 4    | party2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party2 | t2-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_CANCELLED |
      | party2 | t2-s-2    | STATUS_CANCELLED |
      | party1 | t1-b-3    | STATUS_CANCELLED |
      | party2 | t2-s-3    | STATUS_FILLED    |
      | party5 | t5-s-1    | STATUS_FILLED    |
      | party6 | t6-b-1    | STATUS_CANCELLED |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party2 | -8     | 0              | 0            |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 11000 |  1     |
      | sell | 6100  | 15     |
      | buy  | 5900  | 16     |
      | buy  | 1000  |  1     |

    # party2_maintenance:= 8*10000*0.1 + 8*max(0, 6100-10000) = 8000 + 0 = 8000
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party2 | ETH/DEC19 |  8000       |  8800  |  9600   | 11200   |
      | party1 | ETH/DEC19 | 45900       | 50490  | 55080   | 64260   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | BTC   | ETH/DEC19 |  9600  | 99990400 |
      | party1 | BTC   | ETH/DEC19 | 55080  | 99944920 |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | ref price |
      | 10000      | TRADING_MODE_CONTINUOUS | 5       | 9997      | 10002     | 10000     |

  @OpenIP
  Scenario: Same test as above, but without the initial indicative price/volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # Start opening auction with some dead time...
    When the network moves ahead "1" blocks
    # place orders to set the actual price point at which we'll uncross to be 10000
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
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | BTC   | ETH/DEC19 | 13440  | 99986560 |
      | party2 | BTC   | ETH/DEC19 | 13081  | 99986919 |
    # moves forwards several blocks
    When the opening auction period ends for market "ETH/DEC19"
    ## We're seeing these events twice for some reason
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 3    | party2 |
      | party1 | 10000 | 2    | party2 |
      | party1 | 10000 | 3    | party2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    ## Network for distressed party1 -> cancelled, nothing on the book is remaining
    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party2 | t2-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_CANCELLED |
      | party2 | t2-s-2    | STATUS_CANCELLED |
      | party1 | t1-b-3    | STATUS_CANCELLED |
      | party2 | t2-s-3    | STATUS_FILLED    |

    When the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party2 | ETH/DEC19 | 8000        | 9600    |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | BTC   | ETH/DEC19 |  9600  | 99990400 |
      | party1 | BTC   | ETH/DEC19 | 48960  | 99951040 |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | ref price |
      | 10000      | TRADING_MODE_CONTINUOUS | 5       | 9997      | 10002     | 10000     |

  @OpenIPT
  Scenario: Simple test with different indicative price before auction uncross
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | party5 | BTC   | 100000000 |
      | party6 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # Start market with some dead time
    When the network moves ahead "3" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    # Ensure an indicative price/volume of 10, although we will not uncross at this price point
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC19 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GFA | t5-s-1    |
      | party6 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GFA | t6-b-1    |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | party1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | party2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 11200       | 12320  | 13440   | 15680   |
      | party2 | ETH/DEC19 | 10901       | 11991  | 13081   | 15261   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | BTC   | ETH/DEC19 | 13440  | 99986560 |
      | party2 | BTC   | ETH/DEC19 | 13081  | 99986919 |
    When the opening auction period ends for market "ETH/DEC19"
    ## We're seeing these events twice for some reason
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 1    | party5 |
      | party1 | 10000 | 3    | party2 |
      | party1 | 10000 | 1    | party2 |
      | party1 | 10000 | 4    | party2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party2 | t2-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_CANCELLED |
      | party2 | t2-s-2    | STATUS_CANCELLED |
      | party1 | t1-b-3    | STATUS_CANCELLED |
      | party2 | t2-s-3    | STATUS_FILLED    |
      | party5 | t5-s-1    | STATUS_FILLED    |
      | party6 | t6-b-1    | STATUS_CANCELLED |

    When the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party2 | ETH/DEC19 | 8000        | 9600    |
      | party1 | ETH/DEC19 | 45900       | 55080   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | BTC   | ETH/DEC19 |  9600  | 99990400 |
      | party1 | BTC   | ETH/DEC19 | 55080  | 99944920 |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party2 | -8     | 0              | 0            |
      | party1 |  9     | 0              | 0            |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | ref price |
      | 10000      | TRADING_MODE_CONTINUOUS | 5       | 9997      | 10002     | 10000     |

  @OpenIPO
  Scenario: Same again, but higher indicative price
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | BTC   | 100000000 |
      | party2 | BTC   | 100000000 |
      | party3 | BTC   | 100000000 |
      | party4 | BTC   | 100000000 |
      | party5 | BTC   | 100000000 |
      | party6 | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # Start market with some dead time
    When the network moves ahead "3" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    # Ensure an indicative price/volume of 10, although we will not uncross at this price point
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC19 | sell | 1      | 10900 | 0                | TYPE_LIMIT | TIF_GFA | t5-s-1    |
      | party6 | ETH/DEC19 | buy  | 1      | 10900 | 0                | TYPE_LIMIT | TIF_GFA | t6-b-1    |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the parties cancel the following orders:
      | party  | reference |
      | party5 | t5-s-1    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party4 | ETH/DEC19 | sell | 1      | 11000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC19 | sell | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
      | party1 | ETH/DEC19 | buy  | 5      | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC19 | sell | 5      | 10001 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-2    |
      | party1 | ETH/DEC19 | buy  | 4      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t1-b-3    |
      | party2 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GFA | t2-s-3    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party2 | ETH/DEC19 | 10901       | 11991  | 13081   | 15261   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | BTC   | ETH/DEC19 | 13440  | 99986560 |
      | party2 | BTC   | ETH/DEC19 | 13081  | 99986919 |
    # values before uint
    #| party1 | BTC   | ETH/DEC19 | 30241  | 99969759 |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode            |
      | TRADING_MODE_CONTINUOUS |
    Then debug trades
    ## We're seeing these events twice for some reason
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10000 | 3    | party2 |
      | party1 | 10000 | 2    | party2 |
      | party1 | 10000 | 2    | party2 |
    And the mark price should be "10000" for the market "ETH/DEC19"
    ## Network for distressed party1 -> cancelled, nothing on the book is remaining
    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | t1-b-1    | STATUS_FILLED    |
      | party2 | t2-s-1    | STATUS_FILLED    |
      | party1 | t1-b-2    | STATUS_CANCELLED |
      | party2 | t2-s-2    | STATUS_CANCELLED |
      | party1 | t1-b-3    | STATUS_CANCELLED |
      | party2 | t2-s-3    | STATUS_FILLED    |
      | party5 | t5-s-1    | STATUS_CANCELLED |
      | party6 | t6-b-1    | STATUS_FILLED    |

    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | BTC   | ETH/DEC19 |  9600  | 99990400 |
      | party1 | BTC   | ETH/DEC19 | 42840  | 99957160 |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | ref price |
      | 10000      | TRADING_MODE_CONTINUOUS | 5       | 9997      | 10002     | 10000     |

