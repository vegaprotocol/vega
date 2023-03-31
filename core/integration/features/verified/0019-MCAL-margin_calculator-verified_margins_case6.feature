Feature: CASE-6: Trader submits short order that will trade - new formula & zero side of order book (0019-MCAL-001, 0019-MCAL-002)
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
      | aux        | ETH   | 10000000000 |
      | aux2       | ETH   | 10000000000 |
      | lpprov     | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price    | resulting trades | type       | tif     | reference      |
      | aux   | ETH/DEC19 | buy  | 20     | 7900000  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-buy  |
      | aux   | ETH/DEC19 | sell | 20     | 26000000 | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-sell |
      | aux   | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1        |
      | aux2  | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1        |
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
      | party     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM | ETH/DEC19 | buy  | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM | ETH/DEC19 | buy  | 20     | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | buySideMM | ETH/DEC19 | buy  | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the parties cancel the following orders:
      | party | reference     |
      | aux   | cancel-me-buy |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

  Scenario:
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # placing test order
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 13     | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "party1" should have general account balance of "818000000" for asset "ETH"

    And the following trades should be executed:
      | buyer     | price    | size | seller |
      | buySideMM | 25000000 | 13   | party1 |

    Then the following transfers should happen:
      | from   | to        | from account            | to account          | market id | amount   | asset |
      | market | buySideMM | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 14700000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 182000000 | 818000000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 45500000    | 145600000 | 182000000 | 227500000 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | 0              | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 260
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 26000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 26000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to        | from account            | to account          | market id | amount   | asset |
      | market | buySideMM | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 14000000 | ETH   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 135200000 | 851800000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 33800000    | 108160000 | 135200000 | 169000000 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | -13000000      | 0            |

    # CLOSEOUT ATTEMPT (FAILED, no sell-side in order book) BY TRADER
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 13     | 8000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general   |
      | party1 | ETH   | ETH/DEC19 | 135200000 | 851800000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release   |
      | party1 | ETH/DEC19 | 33800000    | 108160000 | 135200000 | 169000000 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -13    | -13000000      | 0            |
