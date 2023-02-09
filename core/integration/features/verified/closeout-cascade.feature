Feature: Closeout-cascades
  # This is a test case to demonstrate that closeout cascade does NOT happen. In this test case, trader3 gets closed
  # out first after buying volume 50 at price 100, and then trader3's position is sold (via the network counterparty) to trader2 whose order is first in the order book
  # (volume 50 price 50)
  # At this moment, trader2 is under margin but the fuse breaks (if the mark price is actually updated from 100 to 50)
  # however, the design of the system is like a fuse and circuit breaker, the mark price will NOT update, so trader2 will not be closed out
  # till a new trade happens, and new mark price is set.
  Background:
    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @NetworkParty
  @CloseOutTrades
  Scenario: Distressed position gets taken over by another party whose margin level is insufficient to support it (however mark price doesn't get updated on closeout trade and hence no further closeouts are carried out) (0005-COLL-002)
    # setup accounts, we are trying to closeout trader3 first and then trader2

    Given the insurance pool balance should be "0" for the market "ETH/DEC19"

    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | BTC   | 1000000000000 |
      | auxiliary2 | BTC   | 1000000000000 |
      | trader2    | BTC   | 2000          |
      | trader3    | BTC   | 100           |
      | lpprov     | BTC   | 1000000000000 |

    Then the cumulated balance for all accounts should be worth "3000000002100"

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp1 | lpprov | ETH/DEC19 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendment  |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    # trading happens at the end of the open auction period
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC19 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the auction ends with a traded volume of "10" at a price of "10"
    And the mark price should be "10" for the market "ETH/DEC19"

    And the cumulated balance for all accounts should be worth "3000000002100"

    # setup trader2 position to be ready to takeover trader3's position once trader3 is closed out
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader2 | ETH/DEC19 | buy  | 50     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-position-3 |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 50          | 75     | 100     | 150     |
    # margin_trader2: 50*10*0.1=50

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | BTC   | ETH/DEC19 | 100    | 1900    |

    And the cumulated balance for all accounts should be worth "3000000002100"

    # setup trader3 position and close it out
    When the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | trader3    | ETH/DEC19 | buy  | 50     | 100   | 0                | TYPE_LIMIT | TIF_GTC | buy-position-3  |
      | auxiliary1 | ETH/DEC19 | sell | 50     | 100   | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | auxiliary2 | ETH/DEC19 | buy  | 50     | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1050  | 96     |
      | sell | 1000  | 10     |
      | buy  | 50    | 50     |
      | buy  | 10    | 50     |
      | buy  | 5     | 5      |
      | buy  | 1     | 100000 |
    And the network moves ahead "1" blocks
    Then the mark price should be "100" for the market "ETH/DEC19"

    # trader3 got closed-out, and trader2 got close-out from the trade with network to close-out trader3, auxiliary2 was in the trade to close-out trader2
    And the following trades should be executed:
      | buyer      | price | size | seller     |
      | auxiliary2 | 10    | 10   | auxiliary1 |
      | trader3    | 100   | 50   | auxiliary1 |
      | trader2    | 50    | 50   | network    |
      | network    | 50    | 50   | trader3    |
      | auxiliary2 | 10    | 50   | network    |
      | network    | 10    | 50   | trader2    |

    And the cumulated balance for all accounts should be worth "3000000002100"
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1005  | 100    |
      | sell | 1000  | 10     |
      | buy  | 5     | 5      |
      | buy  | 1     | 100000 |
    Then the mark price should be "100" for the market "ETH/DEC19"

    # check that trader3 is closed-out but trader2 is not
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | trader2 | ETH/DEC19 | 0           | 0      | 0       | 0       |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader2 | 0      | 0              | -2000        |
      | trader3 | 0      | 0              | -100         |

    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general      |
      | trader2    | BTC   | ETH/DEC19 | 0      | 0            |
      | trader3    | BTC   | ETH/DEC19 | 0      | 0            |
      | auxiliary1 | BTC   | ETH/DEC19 | 114320 | 999999884775 |
      | auxiliary2 | BTC   | ETH/DEC19 | 13180  | 999999989816 |

    # setup new mark price, which is the same as when trader2 traded with network
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 10    | 1                | TYPE_LIMIT | TIF_GTC | aux-s-1   |

    # close-out trade price is not counted as mark price
    And the mark price should be "100" for the market "ETH/DEC19"
    And then the network moves ahead "10" blocks
    And the mark price should be "10" for the market "ETH/DEC19"

    #trader2 and trader3 are still close-out
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader2 | 0      | 0              | -2000        |
      | trader3 | 0      | 0              | -100         |

