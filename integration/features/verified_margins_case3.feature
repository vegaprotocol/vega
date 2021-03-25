Feature: CASE-3: Trader submits long order that will trade - new formula & zero side of order book
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
      | aux        | ETH   | 1000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type        | tif     | reference      |
      | aux     | ETH/DEC19 | buy  | 1      | 7900000  | 0                | TYPE_LIMIT  | TIF_GTC | cancel-me-buy  |
      | aux     | ETH/DEC19 | sell | 1      | 25000000 | 0                | TYPE_LIMIT  | TIF_GTC | cancel-me-sell |

    # setting mark price
    And traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # setting order book
    And traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then traders cancels the following orders reference:
      | trader | reference      |
      | aux    | cancel-me-sell |

    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

  Scenario:
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    # placing test order
    When traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And "trader1" general account for asset "ETH" balance is "542800000"
    And executed trades:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 462800000 | 542800000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 115700000   | 370240000 | 462800000 | 578500000 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 5600000        | 0            |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 160
    When traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 16000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 16000000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the following transfers happened:
      | from   | to      | from account            | to account          | market id | amount   | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 26000000 | ETH   |

    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 488800000 | 542800000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 146900000   | 470080000 | 587600000 | 734500000 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 31600000       | 0            |

    # CLOSEOUT ATTEMPT (FAILED, no buy-side in order book) BY TRADER
    When traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 13     | 8000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 587600000 | 444000000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 146900000   | 470080000 | 587600000 | 734500000 |
    And traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 13     | 31600000       | 0            |
