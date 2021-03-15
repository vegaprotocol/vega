Feature: CASE-2: Trader submits long order that will trade - new formula & low exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | tau/short | lamd/long | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.1       | 0.2       | 0              | 0               | 0     | 5              | 4              | 3.2           | 9400000          | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value   |
      | prices.ETH.value | 9400000 |
    And the traders make the following deposits on asset's general account:
      | trader     | asset | amount     |
      | trader1    | ETH   | 1000000000 |
      | sellSideMM | ETH   | 1000000000 |
      | buySideMM  | ETH   | 1000000000 |
    # setting mark price
    And traders place following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC |


    # setting order book
    And traders place following orders with references:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0      | TYPE_LIMIT | TIF_GTC | _sell1    |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0      | TYPE_LIMIT | TIF_GTC | _sell2    |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0      | TYPE_LIMIT | TIF_GTC | _sell3    |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0      | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0      | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 15     | 8000000  | 0      | TYPE_LIMIT | TIF_GTC | _buy3     |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7700000  | 0      | TYPE_LIMIT | TIF_GTC | _buy4     |


  Scenario:
    # MAKE TRADES
    # no margin account created for trader1, just general account
    And "trader1" has only one account per asset
    # placing test order
    Then traders place following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 13     | 15000000 | 2      | TYPE_LIMIT | TIF_GTC |
    And "trader1" general account for asset "ETH" balance is "575199952"
    And executed trades:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 430400048 | 575199952 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 107600012   | 344320038 | 430400048 | 538000060 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancel the following orders:
      | trader    | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 430400048 | 575199952 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 107600012   | 344320038 | 430400048 | 538000060 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 100
    Then traders place following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC |

    And the following transfers happened:
      | from    | to     | from account        | to account              | market id | amount   | asset |
      | trader1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 52000000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 208000000 | 745600000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 52000000    | 166400000 | 208000000 | 260000000 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | -46400000      | 0            |

    # PARTIAL CLOSEOUT BY TRADER
    Then traders place following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 10     | 8000000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin   | general   |
      | trader1 | ETH   | ETH/DEC19 | 19200000 | 908400000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search   | initial  | release  |
      | trader1 | ETH/DEC19 | 4800000     | 15360000 | 19200000 | 24000000 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 3      | -16707692      | -55692308    |

    # FULL CLOSEOUT BY TRADER
    Then traders place following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 3      | 7000000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general   |
      | trader1 | ETH   | ETH/DEC19 | 0      | 927600000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | -72400000    |
