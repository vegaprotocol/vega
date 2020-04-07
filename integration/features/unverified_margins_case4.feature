Feature: CASE-4: Trader submits short order that will trade - new formula & high exit price
# https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-
# There are discrepancies between the margin values in the spreadsheet and this test case, they need to be verified
# Test end result is the same though

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | tau/long | lamd/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | BTC      | ETH       | ETH   |        94 | simple     |      0.1 |        0.2 |  0 | 0 |     0 |              5 |              4 |           3.2 |              94 |
    And the following traders:
      | name       | amount |
      | trader1    | 10000  |
      | sellSideMM | 10000  |
      | buySideMM  | 10000  |
    # setting mark price
    And traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell |      1 |   103 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   103 |      1 | LIMIT | GTC |


    # setting order book
    And traders place following orders with references:
      | trader     | market id | type | volume | price | trades | type  | tif | reference |
      | sellSideMM | ETH/DEC19 | sell |     10 |   150 |      0 | LIMIT | GTC |     sell1 |
      | sellSideMM | ETH/DEC19 | sell |     14 |   140 |      0 | LIMIT | GTC |     sell2 |
      | sellSideMM | ETH/DEC19 | sell |      2 |   112 |      0 | LIMIT | GTC |     sell3 |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   100 |      0 | LIMIT | GTC |      buy1 |
      | buySideMM  | ETH/DEC19 |  buy |      3 |    96 |      0 | LIMIT | GTC |      buy2 |
      | buySideMM  | ETH/DEC19 |  buy |      9 |    90 |      0 | LIMIT | GTC |      buy3 |
      | buySideMM  | ETH/DEC19 |  buy |     50 |    87 |      0 | LIMIT | GTC |      buy4 |


  Scenario:
    # MAKE TRADES
    Given I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    # no margin account created for trader1, just general account
    And "trader1" have only one account per asset
    # placing test order
    Then traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | trader1    | ETH/DEC19 | sell |     13 |    90 |      3 | LIMIT | GTC |
    And "trader1" general account for asset "ETH" balance is "6752"
    And executed trades:
      |  buyer    | price | size |  seller |
      | buySideMM |   100 |    1 | trader1 |
      | buySideMM |    96 |    3 | trader1 |
      | buySideMM |    90 |    9 | trader1 |
      
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |     28 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   3276 |    6752 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         819 |   2620 |    3276 |    4095 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |    -13 |            28 |           0 |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancels the following orders reference:
      | trader     | reference |
      | buySideMM  |      buy4 |
      | sellSideMM |     sell1 |
      | sellSideMM |     sell2 |
      | sellSideMM |     sell3 |
    And traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | buySideMM  | ETH/DEC19 |  buy |     45 |    70 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |     50 |    75 |      0 | LIMIT | GTC |
      | sellSideMM | ETH/DEC19 | sell |     10 |   100 |      0 | LIMIT | GTC |
      | sellSideMM | ETH/DEC19 | sell |     14 |    88 |      0 | LIMIT | GTC |
      | sellSideMM | ETH/DEC19 | sell |      2 |    84 |      0 | LIMIT | GTC |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   3276 |    6752 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         819 |   2620 |    3276 |    4095 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |    -13 |            28 |           0 |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 80
    Then traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell |      1 |    80 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      1 |    80 |      1 | LIMIT | GTC |

    # MTM
    And the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    130 | ETH   |
    
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   1196 |    8962 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         299 |    956 |    1196 |    1495 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |    -13 |           158 |           0 |

  # FULL CLOSEOUT BY TRADER
  Then traders place following orders:
    | trader  | market id | type | volume | price | trades | type  | tif |
    | trader1 | ETH/DEC19 | buy |      13 |    90 |      2 | LIMIT | GTC |
  And position API produce the following:
    | trader  | volume | unrealisedPNL | realisedPNL |
    | trader1 |      0 |             0 |          62 |