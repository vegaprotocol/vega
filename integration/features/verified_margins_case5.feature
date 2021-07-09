Feature: CASE-5: Trader submits short order that will trade - new formula & low exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value   |
      | prices.ETH.value | 9400000 |
    And the traders deposit on asset's general account the following amount:
      | trader     | asset | amount       |
      | trader1    | ETH   | 980000000    |
      | sellSideMM | ETH   | 100000000000 |
      | buySideMM  | ETH   | 100000000000 |
      | aux        | ETH   | 1000000000   |
      | aux2       | ETH   | 1000000000   |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price    | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 6999999  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 50000001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "10300000" for the market "ETH/DEC19"

    # setting mark price
    And the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    # setting order book
    And the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 10     | 15000000 | 0                | TYPE_LIMIT | TIF_GTC | sell1     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | sell2     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | sell3     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 9      | 9000000  | 0                | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC | buy4      |


  Scenario:
    # no margin account created for trader1, just general account
    And "trader1" should have one account per asset
    # placing test order
    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 13     | 9000000 | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "trader1" should have general account balance of "698400040" for asset "ETH"
    And the following trades should be executed:
      | buyer     | price    | size | seller  |
      | buySideMM | 10000000 | 1    | trader1 |
      | buySideMM | 9600000  | 3    | trader1 |
      | buySideMM | 9000000  | 9    | trader1 |
    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 2800000 | ETH   |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 284399960 | 698400040 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 71099990    | 227519968 | 284399960 | 355499950 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | 2800000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the traders cancel the following orders:
      | trader     | reference |
      | buySideMM  | buy4      |
      | sellSideMM | sell2     |
      | sellSideMM | sell3     |
    And the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM  | ETH/DEC19 | buy  | 45     | 7000000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7500000  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 8000000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 284399960 | 698400040 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 71099990    | 227519968 | 284399960 | 355499950 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | 2800000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 300
    When the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 50     | 30000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 27     | 30000000 | 4                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # MTM
    And the following transfers should happen:
      | from    | to      | from account         | to account              | market id | amount    | asset |
      | trader1 | market  | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 273000000 | ETH   |
      | trader1 | trader1 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 144600040 | ETH   |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 156000000 | 553800000 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 39000000    | 124800000 | 156000000 | 195000000 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | -270200000     | 0            |

    # ENTER SEARCH LEVEL (& DEPLEAT GENERAL ACCOUNT)
    When the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 11     | 50000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 50000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 65000000    | 208000000 | 260000000 | 325000000 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | -530200000     | 0            |

    # FORCED CLOSEOUT
    When the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 21     | 80000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 11     | 80000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 0       |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | -980000000   |
