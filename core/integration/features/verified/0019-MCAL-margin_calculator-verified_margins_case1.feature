Feature: CASE-1: Trader submits long order that will trade - new formula & high exit price (0019-MCAL-001, 0019-MCAL-002, 0019-MCAL-003)
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount     |
      | party1     | ETH   | 1000000000 |
      | sellSideMM | ETH   | 2000000000 |
      | buySideMM  | ETH   | 2000000000 |
      | aux        | ETH   | 1000000000 |
      | aux2       | ETH   | 1000000000 |
      | lpprov     | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 400 | 1 | buy  | BID | 400 | 10 |
      | lpprov | ETH/DEC19 | 600 | 1 | sell | ASK | 600 | 10 |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price    | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC19" should be:
      | target stake | supplied stake |
      | 20600000     | 900000000      |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "10300000" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setting mark price
    And the parties place the following orders:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    # setting order book
    And the parties place the following orders:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | _sell1    |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | _sell2    |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | _sell3    |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 15     | 9000000  | 0                | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC | _buy4     |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | buy  | 1        | 1      |
      | buy  | 8700000  | 50     |
      | buy  | 9000000  | 15     |
      | buy  | 9600000  | 3      |
      | buy  | 9999990  | 400    |
      | buy  | 10000000 | 1      |
      | sell | 11200000 | 2      |
      | sell | 11200010 | 600    |
      | sell | 14000000 | 11     |
      | sell | 20000000 | 1      |
      | sell | 25000000 | 100    |

  Scenario:
    # no margin account created for party1, just general account
    Given "party1" should have one account per asset
    # placing test order
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "party1" should have general account balance of "821118876" for asset "ETH"
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | buy  | 1        | 1      |
      | buy  | 8700000  | 50     |
      | buy  | 9000000  | 15     |
      | buy  | 9600000  | 3      |
      | buy  | 9999990  | 400    |
      | buy  | 10000000 | 1      |
      | sell | 11200000 | 0      |
      | sell | 11200010 | 0      |
      | sell | 14000000 | 11     |
      | sell | 14000010 | 589    |
      | sell | 20000000 | 1      |
      | sell | 25000000 | 100    |
    And the following trades should be executed:
      | buyer  | price    | size | seller     |
      | party1 | 11200000 | 2  | sellSideMM |
      | party1 | 11200010 | 11 | lpprov     |

    Then the parties should have the following profit and loss:
      | party      | volume | unrealised pnl | realised pnl |
      | party1     | 13     | 20             | 0            |
      | sellSideMM | -3     | -900030        | 0            |
      | lpprov     | -11    | 0              | 0            |

    Then the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount  | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 20 | ETH |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH | ETH/DEC19 | 178881144 | 821118876 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 44720286 | 143104915 | 178881144 | 223601430 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13 | 20 | 0 |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the parties cancel the following orders:
      | party     | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |
      | buySideMM | buy3      |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | buy  | 1        | 1      |
      | buy  | 8699990  | 400    |
      | buy  | 8700000  | 50     |
      | buy  | 9000000  | 0      |
      | buy  | 9600000  | 0      |
      | buy  | 9999990  | 0      |
      | buy  | 10000000 | 0      |
      | sell | 11200000 | 0      |
      | sell | 11200010 | 0      |
      | sell | 14000000 | 11     |
      | sell | 14000010 | 589    |
      | sell | 20000000 | 1      |
      | sell | 25000000 | 100    |
    When the parties place the following orders:
      | party     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM | ETH/DEC19 | buy | 1  | 11900000 | 0 | TYPE_LIMIT | TIF_GTC | ref-1 |
      | buySideMM | ETH/DEC19 | buy | 3  | 11800000 | 0 | TYPE_LIMIT | TIF_GTC | ref-2 |
      | buySideMM | ETH/DEC19 | buy | 15 | 11700000 | 0 | TYPE_LIMIT | TIF_GTC | ref-3 |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH | ETH/DEC19 | 178881144 | 821118876 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 44720286 | 143104915 | 178881144 | 223601430 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13 | 20 | 0 |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 200
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 20000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount   | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 36399870 | ETH |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH | ETH/DEC19 | 215281014 | 821118876 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 65800007 | 210560022 | 263200028 | 329000035 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13 | 36399890 | 0 |

    # FULL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 13 | 11600000 | 3 | TYPE_LIMIT | TIF_GTC |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0 | 0 | 6999890 |
