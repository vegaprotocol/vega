Feature: CASE-1: Trader submits long order that will trade - new formula & high exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
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
      | trader  | market id | side | volume | price    | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1        | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT  | TIF_GTC | 
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
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0      | TYPE_LIMIT | TIF_GTC | _sell1    |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0      | TYPE_LIMIT | TIF_GTC | _sell2    |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0      | TYPE_LIMIT | TIF_GTC | _sell3    |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0      | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0      | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 15     | 9000000  | 0      | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0      | TYPE_LIMIT | TIF_GTC | _buy4     |


  Scenario:
    # no margin account created for trader1, just general account
    And "trader1" should have one account per asset
    # placing test order
    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "trader1" should have general account balance of "611199968" for asset "ETH"
    And the following trades should be executed:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 394400032 | 611199968 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then the traders cancel the following orders:
      | trader    | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |
      | buySideMM | buy3      |
    When the traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM | ETH/DEC19 | buy  | 1      | 19000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM | ETH/DEC19 | buy  | 3      | 18000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | buySideMM | ETH/DEC19 | buy  | 15     | 17000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 394400032 | 611199968 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 200
    When the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 20000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers should happen:
      | from   | to      | from account            | to account          | market id | amount   | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 78000000 | ETH   |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 344000020 | 739599980 |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 86000005    | 275200016 | 344000020 | 430000025 |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 83600000       | 0            |

    # FULL CLOSEOUT BY TRADER
    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 13     | 16500000 | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the traders should have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | 49600000     |
