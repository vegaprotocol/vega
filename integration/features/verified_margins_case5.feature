Feature: CASE-5: Trader submits short order that will trade - new formula & low exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | tau/short | lamd/long | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations |
      | ETH/DEC19 | BTC      | ETH       | ETH   |   9400000 | simple     |       0.1 |       0.2 |  0 | 0 |     0 |              5 |              4 |           3.2 |          940000 |           0 | continous    |        0 |                 0 |            0 |                 0  |                |             |                 |
    And the following traders:
      | name       | amount       |
      | trader1    | 1000000000   |
      | sellSideMM | 100000000000 |
      | buySideMM  | 100000000000 |
    # setting mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1      | TYPE_LIMIT | TIF_GTC |


    # setting order book
    And traders place following orders with references:
      | trader     | market id | type | volume | price    | trades | type  | tif | reference |
      | sellSideMM | ETH/DEC19 | sell | 10     | 15000000 | 0      | TYPE_LIMIT | TIF_GTC | sell1     |
      | sellSideMM | ETH/DEC19 | sell | 14     | 14000000 | 0      | TYPE_LIMIT | TIF_GTC | sell2     |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0      | TYPE_LIMIT | TIF_GTC | sell3     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0      | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0      | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 9      | 9000000  | 0      | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0      | TYPE_LIMIT | TIF_GTC | buy4      |


  Scenario:
    # MAKE TRADES
    Given I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    # no margin account created for trader1, just general account
    And "trader1" have only one account per asset
    # placing test order
    Then traders place following orders:
      | trader  | market id | type | volume | price   | trades | type  | tif |
      | trader1 | ETH/DEC19 | sell | 13     | 9000000 | 3      | TYPE_LIMIT | TIF_GTC |
    And "trader1" general account for asset "ETH" balance is "671600040"
    And executed trades:
      | buyer     | price    | size | seller  |
      | buySideMM | 10000000 | 1    | trader1 |
      | buySideMM | 9600000  | 3    | trader1 |
      | buySideMM | 9000000  | 9    | trader1 |
    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 2800000 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 331199960 | 671600040 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 82799990    | 264959968 | 331199960 | 413999950 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -13    | 2800000       | 0           |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancels the following orders reference:
      | trader     | reference |
      | buySideMM  | buy4      |
      | sellSideMM | sell2     |
      | sellSideMM | sell3     |
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | buySideMM  | ETH/DEC19 | buy  | 45     | 7000000  | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 7500000  | 0      | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 14     | 10000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 2      | 8000000  | 0      | TYPE_LIMIT | TIF_GTC |
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 331199960 | 671600040 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 82799990    | 264959968 | 331199960 | 413999950 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -13    | 2800000       | 0           |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 300
    Then traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 50     | 30000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 27     | 30000000 | 4      | TYPE_LIMIT | TIF_GTC |

    # MTM
    And the following transfers happened:
      | from    | to      | fromType             | toType                  | id        | amount    | asset |
      | trader1 | market  | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 273000000 | ETH   |
      | trader1 | trader1 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 253800040 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 312000000 | 417800000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 78000000    | 249600000 | 312000000 | 390000000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -13    | -270200000    | 0           |

    # ENTER SEARCH LEVEL (& DEPLEAT GENERAL ACCOUNT)
    Then traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 11     | 50000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 50000000 | 2      | TYPE_LIMIT | TIF_GTC |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 130000000   | 416000000 | 520000000 | 650000000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -13    | -530200000    | 0           |

    # FORCED CLOSEOUT
    Then traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 21     | 80000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 11     | 80000000 | 2      | TYPE_LIMIT | TIF_GTC |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 0      | 0             | -1000000000 |
