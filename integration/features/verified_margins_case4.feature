Feature: CASE-4: Trader submits short order that will trade - new formula & high exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | tau/short | lamd/long | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.1       | 0.2       | 0              | 0               | 0     | 5              | 4              | 3.2           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
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
      | sellSideMM | ETH/DEC19 | sell | 10     | 15000000 | 0      | TYPE_LIMIT | TIF_GTC | sell1     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 14000000 | 0      | TYPE_LIMIT | TIF_GTC | sell2     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0      | TYPE_LIMIT | TIF_GTC | sell3     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0      | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0      | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 9      | 9000000  | 0      | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0      | TYPE_LIMIT | TIF_GTC | buy4      |


  Scenario:
    # no margin account created for trader1, just general account
    And "trader1" has only one account per asset
    # placing test order
    Then traders place following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 13     | 9000000 | 3      | TYPE_LIMIT | TIF_GTC |
    And "trader1" general account for asset "ETH" balance is "718400040"
    And executed trades:
      | buyer     | price    | size | seller  |
      | buySideMM | 10000000 | 1    | trader1 |
      | buySideMM | 9600000  | 3    | trader1 |
      | buySideMM | 9000000  | 9    | trader1 |

    Then the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 2800000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 284399960 | 718400040 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 71099990    | 227519968 | 284399960 | 355499950 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | 2800000        | 0            |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    And traders cancel the following orders:
      | trader     | reference |
      | buySideMM  | buy4      |
      | sellSideMM | sell1     |
      | sellSideMM | sell2     |
      | sellSideMM | sell3     |
    And traders place following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     |
      | buySideMM  | ETH/DEC19 | buy  | 45     | 7000000  | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7500000  | 0      | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 10     | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 14     | 8800000  | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 2      | 8400000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 284399960 | 718400040 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 71099990    | 227519968 | 284399960 | 355499950 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | 2800000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 80
    Then traders place following orders:
      | trader     | market id | side | volume | price   | resulting trades | type       | tif     |
      | sellSideMM | ETH/DEC19 | sell | 1      | 8000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 8000000 | 1                | TYPE_LIMIT | TIF_GTC |

    # MTM
    And the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount   | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 13000000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin   | general   |
      | trader1 | ETH   | ETH/DEC19 | 79999972 | 935800028 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search   | initial  | release  |
      | trader1 | ETH/DEC19 | 19999993    | 63999977 | 79999972 | 99999965 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | -13    | 15800000       | 0            |

    # FULL CLOSEOUT BY TRADER
    Then traders place following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 13     | 9000000 | 2                | TYPE_LIMIT | TIF_GTC |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | 6200000      |
