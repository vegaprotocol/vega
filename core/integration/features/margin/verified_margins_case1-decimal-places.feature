Feature: CASE-1: Trader submits long order that will trade - new formula & high exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2              | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount          |
      | party1     | ETH   | 100000000000000 |
      | sellSideMM | ETH   | 100000000000000 |
      | buySideMM  | ETH   | 100000000000000 |
      | aux        | ETH   | 100000000000000 |
      | aux2       | ETH   | 100000000000000 |
      | lpprov     | ETH   | 100000000000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | sell | ASK              | 50         | 100    | submission |
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
    And the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    # setting order book
    And the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | _sell1    |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | _sell2    |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | _sell3    |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 15     | 9000000  | 0                | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC | _buy4     |


  Scenario:
    # no margin account created for party1, just general account
    And "party1" should have one account per asset
    # placing test order
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "party1" should have general account balance of "99611199999980" for asset "ETH"
    And the following trades should be executed:
      | buyer  | price    | size | seller     |
      | party1 | 11200000 | 2    | sellSideMM |
      | party1 | 14000000 | 11   | sellSideMM |

    Then the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount     | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin       | general        |
      | party1 | ETH   | ETH/DEC19 | 394400000020 | 99611199999980 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search       | initial      | release      |
      | party1 | ETH/DEC19 | 98600000005 | 315520000016 | 394400000020 | 493000000025 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 5600000000     | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the parties cancel the following orders:
      | party     | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |
      | buySideMM | buy3      |
    When the parties place the following orders with ticks:
      | party     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM | ETH/DEC19 | buy  | 1      | 19000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM | ETH/DEC19 | buy  | 3      | 18000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | buySideMM | ETH/DEC19 | buy  | 15     | 17000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin       | general        |
      | party1 | ETH   | ETH/DEC19 | 394400000020 | 99611199999980 |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search       | initial      | release      |
      | party1 | ETH/DEC19 | 98600000005 | 315520000016 | 394400000020 | 493000000025 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 5600000000     | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 200
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 20000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount      | asset |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 78000000000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin       | general        |
      | party1 | ETH   | ETH/DEC19 | 344000000032 | 99739599999968 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search       | initial      | release      |
      | party1 | ETH/DEC19 | 86000000008 | 275200000025 | 344000000032 | 430000000040 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 13     | 83600000000    | 0            |

    # FULL CLOSEOUT BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 13     | 16500000 | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 49600000000  |
