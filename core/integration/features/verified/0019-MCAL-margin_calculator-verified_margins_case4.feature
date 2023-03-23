Feature: CASE-4: Trader submits short order that will trade - new formula & high exit price (0019-MCAL-001, 0019-MCAL-002, 0019-MCAL-0016)
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e0                    | 0                         |
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount     |
      | party1     | ETH   | 1000000000 |
      | sellSideMM | ETH   | 1000000000 |
      | buySideMM  | ETH   | 1000000000 |
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
      | sellSideMM | ETH/DEC19 | sell | 10     | 15000000 | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | sell2     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | sell3     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 9      | 9000000  | 0                | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC | buy4      |


  Scenario:
    # no margin account created for party1, just general account
    Given "party1" should have one account per asset
    # placing test order
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 13     | 9000000 | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "party1" should have general account balance of "718400040" for asset "ETH"
    And the following trades should be executed:
      | buyer     | price    | size | seller |
      | buySideMM | 10000000 | 1    | party1 |
      | buySideMM | 9600000  | 3    | party1 |
      | buySideMM | 9000000  | 9    | party1 |

    Then the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount  | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 2800000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 284399960 | 718400040 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 71099990    | 227519968 | 284399960 | 355499950 |
    #| party1 | ETH/DEC19 | 15100007    | 48320022 | 60400028 | 75500035 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | 2800000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    And the parties cancel the following orders:
      | party      | reference |
      | buySideMM  | buy4      |
      | sellSideMM | sell1     |
      | sellSideMM | sell2     |
      | sellSideMM | sell3     |
    And the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM  | ETH/DEC19 | buy  | 45     | 7000000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7500000  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideMM | ETH/DEC19 | sell | 10     | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 8800000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 8400000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 284399960 | 718400040 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 71099990    | 227519968 | 284399960 | 355499950 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | 2800000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 80
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 8000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 8000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # MTM
    And the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount   | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 13000000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin   | general   |
      | party1 | ETH   | ETH/DEC19 | 79999972 | 935800028 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search   | initial  | release  |
      | party1 | ETH/DEC19 | 19999993    | 63999977 | 79999972 | 99999965 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | 15800000       | 0            |

    # FULL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 13     | 9000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 6200000      |
