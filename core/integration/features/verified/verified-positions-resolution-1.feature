Feature: Position resolution case 1

  Background:

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.BTC.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | ethDec21Oracle     | 0.9145                    | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: close out when there is not enough orders on the orderbook to cover the position (0007-POSN-009, 0008-TRAD-001, 0008-TRAD-002, 0008-TRAD-005)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLooser | BTC   | 11600         |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # insurance pool generation - setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    # insurance pool generation - trade
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    # margin level: vol* slippage = vol * (MarkPrice-ExitPrice) =290 * (150-(1*10+140*1)/11) = 290*137 = 39700

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 11600  | 0       |

    And the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 39781       | 127299 | 159124  | 198905  |

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 290    | 0              | 0            |
      | sellSideProvider | -290   | 0              | 0            |
      | buySideProvider  | 0      | 0              | 0            |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |

    # insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    When the parties place the following orders with ticks:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    # insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # MTM (designatedLooser): 11600-(290*(150-120))=11600-8700=2900
    Then the parties should have the following account balances:
      | party            | asset | market id | margin  | general      |
      | designatedLooser | BTC   | ETH/DEC19 | 2900    | 0            |
      | sellSideProvider | BTC   | ETH/DEC19 | 127740  | 999999880960 |
      | buySideProvider  | BTC   | ETH/DEC19 | 320     | 999999999680 |
      | aux              | BTC   | ETH/DEC19 | 320     | 999999999650 |
      | aux2             | BTC   | ETH/DEC19 | 440     | 999999999590 |

    # margin level: vol* slippage = vol * (MarkPrice-ExitPrice) =290 * (120-(1*10+40*1)/11) = 290*116 = 33640
    And the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 | 31825       | 101840 | 127300  | 159125  |

    # check positions
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 290    | -8700          | 0            |
      | sellSideProvider | -291   | 8700           | 0            |
      | buySideProvider  | 1      | 0              | 0            |
      | aux              | 1      | -30            | 0            |
      | aux2             | -1     | 30             | 0            |

    # checking margins
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 | 2900   | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2021-03-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.BTC.value | 120   |

    And then the network moves ahead "10" blocks

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 290    | -8700          | 0            |
      | sellSideProvider | -291   | 8700           | 0            |
      | buySideProvider  | 1      | 0              | 0            |
      | aux              | 1      | -30            | 0            |
      | aux2             | -1     | 30             | 0            |
