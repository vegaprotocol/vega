Feature: Position resolution case 4

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.5349                 | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: close out when there is enough orders on the orderbook to cover the position (0008-TRAD-006, 0008-TRAD-007))
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 10000         |
      | auxiliary        | BTC   | 1000000000000 |
      | auxiliary2       | BTC   | 1000000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary2 | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | auxiliary  | ETH/DEC19 | sell | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "180" for the market "ETH/DEC19"

    # insurance pool generation - setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

    # insurance pool generation - trade
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | sell | 100    | 180   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the mark price should be "180" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 1      |
      | sell | 200   | 150    |

    # margin level: vol* slippage = vol * (MarkPrice-ExitPrice) =100 * (200-180) = 100*20 = 2000

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 8000   | 2500    |

    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 2000        | 6400   | 8000    | 10000   |

    # insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party            | reference       |
      | sellSideProvider | sell-provider-1 |

    # add back some volume on the sell side
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 150    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |

    # insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "300" for the market "ETH/DEC19"

    #check positions
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -10000       |
      | buySideProvider  | 101    | 11500          | -1363        |

    # checking margins
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"


    # now we check what's left in the orderbook
    # we expect 50 orders to be left there on the sell side
    # we buy a first time 50 to consume the book
    # then try to buy 1 again -> result in no trades -> sell side empty.
    # Try to sell one for low price -> no trades -> buy side empty -> order book empty.
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 350   | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 350   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_FOK | ref-3     |
