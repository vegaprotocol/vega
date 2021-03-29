Feature: CASE-1: Trader submits long order that will trade - new formula & high exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | tau/short | lamd/long | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.1       | 0.2       | 0              | 0               | 0     | 5              | 4              | 3.2           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value   |
      | prices.ETH.value | 9400000 |
    And the traders make the following deposits on asset's general account:
      | trader     | asset | amount     |
      | trader1    | ETH   | 1000000000 |
      | sellSideMM | ETH   | 1000000000 |
      | buySideMM  | ETH   | 1000000000 |
      | aux        | ETH   | 1000000000 |
      | aux2       | ETH   | 1000000000 |
        # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1        | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "10300000"
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    
    # setting mark price
    And traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    # setting order book
    And traders place the following orders:
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
    And "trader1" has only one account per asset
    # placing test order
    When traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "trader1" general account for asset "ETH" balance is "611199968"
    And executed trades:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 394400032 | 611199968 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancel the following orders:
      | trader    | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |
      | buySideMM | buy3      |
    When traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | buySideMM | ETH/DEC19 | buy  | 1      | 19000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM | ETH/DEC19 | buy  | 3      | 18000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | buySideMM | ETH/DEC19 | buy  | 15     | 17000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 394400032 | 611199968 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 200
    When traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 20000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 20000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount   | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 78000000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 344000020 | 739599980 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 86000005    | 275200016 | 344000020 | 430000025 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 83600000       | 0            |

    # FULL CLOSEOUT BY TRADER
    When traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 13     | 16500000 | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | 49600000     |
