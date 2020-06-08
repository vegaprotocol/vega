Feature: CASE-1: Trader submits long order that will trade - new formula & high exit price
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | tau/short | lamd/long | mu | r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | BTC      | ETH       | ETH   | 9400000   | simple     | 0.1      | 0.2        | 0  | 0 | 0     | 5              | 4              | 3.2           | 9400000         |
    And the following traders:
      | name       | amount     |
      | trader1    | 1000000000 |
      | sellSideMM | 1000000000 |
      | buySideMM  | 1000000000 |
    # setting mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1      | TYPE_LIMIT | TIF_GTC |


    # setting order book
    And traders place following orders with references:
      | trader     | market id | type | volume | price    | trades | type  | tif | reference |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0      | TYPE_LIMIT | TIF_GTC | _sell1    |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0      | TYPE_LIMIT | TIF_GTC | _sell2    |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0      | TYPE_LIMIT | TIF_GTC | _sell3    |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 0      | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 3      | 9600000  | 0      | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 15     | 9000000  | 0      | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 50     | 8700000  | 0      | TYPE_LIMIT | TIF_GTC | _buy4     |


  Scenario:
    # MAKE TRADES
    Given I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    # no margin account created for trader1, just general account
    And "trader1" have only one account per asset
    # placing test order
    Then traders place following orders:
      | trader  | market id | type | volume | price    | trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  | 13     | 15000000 | 2      | TYPE_LIMIT | TIF_GTC |
    And "trader1" general account for asset "ETH" balance is "683999968"
    And executed trades:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount  | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 | 5600000 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 321600032 | 683999968 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 80400008    | 257280025 | 321600032 | 402000040 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 13     | 5600000       | 0           |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancels the following orders reference:
      | trader    | reference |
      | buySideMM | buy1      |
      | buySideMM | buy2      |
      | buySideMM | buy3      |
    Then traders place following orders:
      | trader    | market id | type | volume | price    | trades | type  | tif |
      | buySideMM | ETH/DEC19 | buy  | 1      | 19000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM | ETH/DEC19 | buy  | 3      | 18000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM | ETH/DEC19 | buy  | 15     | 17000000 | 0      | TYPE_LIMIT | TIF_GTC |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 321600032 | 683999968 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 80400008    | 257280025 | 321600032 | 402000040 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 13     | 5600000       | 0           |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 200
    Then traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 1      | 20000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 20000000 | 1      | TYPE_LIMIT | TIF_GTC |

    And the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount   | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 | 78000000 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 240000020 | 843599980 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 60000005    | 192000016 | 240000020 | 300000025 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 13     | 83600000      | 0           |

    # FULL CLOSEOUT BY TRADER
    Then traders place following orders:
      | trader  | market id | type | volume | price    | trades | type  | tif |
      | trader1 | ETH/DEC19 | sell | 13     | 16500000 | 3      | TYPE_LIMIT | TIF_GTC |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 0      | 0             | 49600000    |
