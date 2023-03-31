Feature: CASE-2: Trader submits long order that will trade - new formula & low exit price (0019-MCAL-001, 0019-MCAL-002, 0019-MCAL-003)
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount     |
      | party1     | ETH   | 1000000000 |
      | sellSideMM | ETH   | 2000000000 |
      | buySideMM  | ETH   | 2000000000 |
      | aux        | ETH   | 1000000000 |
      | aux2       | ETH   | 1000000000 |
      | lpprov     | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | sell | ASK              | 50         | 10     | submission |
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
    And "party1" should have general account balance of "575199952" for asset "ETH"
    And the following trades should be executed:
      | buyer  | price    | size | seller     |
      | party1 | 11200000 | 2    | sellSideMM |
      | party1 | 14000000 | 11   | sellSideMM |

    Then the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount  | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 430400048 | 575199952 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 107600012   | 344320038 | 430400048 | 538000060 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 5600000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the parties cancel the following orders:
      | party     | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 430400048 | 575199952 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 107600012   | 344320038 | 430400048 | 538000060 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 5600000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 100
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount   | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 52000000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 208000000 | 745600000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 52000000    | 166400000 | 208000000 | 260000000 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | -46400000      | 0            |

    # PARTIAL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 10     | 8000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin   | general   |
      | party1 | ETH   | ETH/DEC19 | 19200000 | 908400000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search   | initial  | release  |
      | party1 | ETH/DEC19 | 4800000     | 15360000 | 19200000 | 24000000 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 3      | -16707692      | -55692308    |

    # FULL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 3      | 7000000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party1 | ETH   | ETH/DEC19 | 0      | 927600000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | -72400000    |
