Feature: CASE-2: Trader submits long order that will trade - new formula & low exit price (0019-MCAL-001, 0019-MCAL-002, 0019-MCAL-003)

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |
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
      | lpprov | ETH/DEC19 | 500 | 1 | buy  | BID | 500 | 10 |
      | lpprov | ETH/DEC19 | 500 | 1 | sell | ASK | 500 | 10 |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price    | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
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
      | buySideMM  | ETH/DEC19 | buy  | 15     | 8000000  | 0                | TYPE_LIMIT | TIF_GTC | _buy3     |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7700000  | 0                | TYPE_LIMIT | TIF_GTC | _buy4     |


  Scenario:
    # MAKE TRADES
    # no margin account created for party1, just general account
    Given "party1" should have one account per asset
    # placing test order
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "party1" should have general account balance of "737919784" for asset "ETH"
    And the following trades should be executed:
      | buyer  | price    | size | seller     |
      | party1 | 11200000 | 2    | sellSideMM |
      | party1 | 11200010 | 11   | lpprov     |

    Then the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount  | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 20 | ETH |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 262080236 | 737919784 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party1 | ETH/DEC19 | 65520059    |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 20             | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the parties cancel the following orders:
      | party     | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 262080236 | 737919784 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party1 | ETH/DEC19 | 65520059    |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 20             | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 100
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount   | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 15600130 | ETH |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH | ETH/DEC19   | 246480106 | 737919784 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | 
      | party1 | ETH/DEC19 | 58500000    | 
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | -15600110      | 0            |

    # PARTIAL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 10     | 8000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin   | general   |
      | party1 | ETH | ETH/DEC19   | 43200000 | 915199890 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party1 | ETH/DEC19 | 10800000    |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 3      | -9600025       | -32000085    |

    # FULL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 3      | 7000000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party1 | ETH | ETH/DEC19 | 0 | 958399890 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0 | 0 | -41600110 |

