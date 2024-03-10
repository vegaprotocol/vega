Feature: CASE-5: Trader submits short order that will trade - new formula & low exit price (0019-MCAL-001, 0019-MCAL-002, 0019-MCAL-003, 0019-MCAL-016)
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount       |
      | party1 | ETH | 310000000 |
      | sellSideMM | ETH   | 100000000000 |
      | buySideMM  | ETH   | 100000000000 |
      | aux        | ETH   | 1000000000   |
      | aux2       | ETH   | 1000000000   |
      | lpprov     | ETH   | 1000000000   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 16 | 1 | buy  | BID | 16 | 10 |
      | lpprov | ETH/DEC19 | 15 | 1 | sell | ASK | 15 | 10 |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price    | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 6999999  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50000001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "10300000" for the market "ETH/DEC19"

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
    And "party1" should have one account per asset
    # placing test order
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 13 | 9000000 | 2 | TYPE_LIMIT | TIF_GTC | ref-1 |
    And "party1" should have general account balance of "193000126" for asset "ETH"


    And the following trades should be executed:
      | buyer     | price    | size | seller |
      | buySideMM | 10000000 | 1    | party1 |
      | lpprov    | 9999990  | 12   | party1 |

    Then the following transfers should happen:
      | from      | to        | from account         | to account          | market id | amount    | asset |
      | party1    | party1    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 53560000  | ETH   |
      | buySideMM | buySideMM | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 412000000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 103999896 | 193000126 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | 10             | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the parties cancel the following orders:
      | party      | reference |
      | buySideMM  | buy4      |
      | sellSideMM | sell2     |
      | sellSideMM | sell3     |
    And the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM  | ETH/DEC19 | buy  | 45     | 7000000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7500000  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | sellSideMM | ETH/DEC19 | sell | 2 | 11000000 | 0 | TYPE_LIMIT | TIF_GTC | ref-4 |
# no MTM yet, so accounts are not changing
    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 103999896 | 193000126 |

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | 10             | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 300
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 50     | 30000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM | ETH/DEC19 | buy | 27 | 30000000 | 2 | TYPE_LIMIT | TIF_GTC | ref-2 |
# MTM
    Then the following transfers should happen:
      | from       | to         | from account            | to account          | market id | amount    | asset |
      | sellSideMM | sellSideMM | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 196339678 | ETH   |
      | market     | buySideMM  | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 180       | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 103999636 | 193000126 |

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | -250           | 0            |
     
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 11     | 50000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM | ETH/DEC19 | buy | 50 | 50000000 | 4 | TYPE_LIMIT | TIF_GTC | ref-2 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 0       |

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0 | 0 | -297000012 |
