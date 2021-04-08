Feature: CASE-3: Trader submits long order that will trade - new formula & zero side of order book
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 1      |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value   |
      | prices.ETH.value | 9400000 |
    And the traders deposit on asset's general account the following amount:
      | trader     | asset | amount     |
      | trader1    | ETH   | 1000000000 |
      | sellSideMM | ETH   | 1000000000 |
      | buySideMM  | ETH   | 1000000000 |
      | aux        | ETH   | 1000000000 |
      | aux2       | ETH   | 1000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type        | tif     | reference      |
      | aux     | ETH/DEC19 | buy  | 1      | 7900000  | 0                | TYPE_LIMIT  | TIF_GTC | cancel-me-buy  |
      | aux     | ETH/DEC19 | sell | 1      | 25000000 | 0                | TYPE_LIMIT  | TIF_GTC | cancel-me-sell |
      | aux     | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-1        |
      | aux2    | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-1        |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "10300000" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # setting mark price
    And the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # setting order book
    And the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the traders cancel the following orders:
      | trader | reference      |
      | aux    | cancel-me-sell |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

  Scenario:
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # placing test order
    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "trader1" should have general account balance of "542800000" for asset "ETH"
    And the following trades should be executed:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 462800000 | 542800000 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 115700000   | 370240000 | 462800000 | 578500000 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 160
    When the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 16000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 16000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount   | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 26000000 | ETH   |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 488800000 | 542800000 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 146900000   | 470080000 | 587600000 | 734500000 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 31600000       | 0            |

    # CLOSEOUT ATTEMPT (FAILED, no buy-side in order book) BY TRADER
    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 13     | 8000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 587600000 | 444000000 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 146900000   | 470080000 | 587600000 | 734500000 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 31600000       | 0            |
