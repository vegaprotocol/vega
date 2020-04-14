Feature: CASE-2: Trader submits long order that will trade - new formula & low exit price
# https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

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
      | sellSideMM | ETH/DEC19 | sell |    100 |   250 |      0 | LIMIT | GTC |    _sell1 |
      | sellSideMM | ETH/DEC19 | sell |     11 |   140 |      0 | LIMIT | GTC |    _sell2 |
      | sellSideMM | ETH/DEC19 | sell |      2 |   112 |      0 | LIMIT | GTC |    _sell3 |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   100 |      0 | LIMIT | GTC |      buy1 |
      | buySideMM  | ETH/DEC19 |  buy |      3 |    96 |      0 | LIMIT | GTC |      buy2 |
      | buySideMM  | ETH/DEC19 |  buy |     15 |    80 |      0 | LIMIT | GTC |     _buy3 |
      | buySideMM  | ETH/DEC19 |  buy |     50 |    77 |      0 | LIMIT | GTC |     _buy4 |


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
      | trader1    | ETH/DEC19 |  buy |     13 |   150 |      2 | LIMIT | GTC |
    And "trader1" general account for asset "ETH" balance is "6468"
    And executed trades:
      |  buyer  | price | size |       seller |
      | trader1 |   112 |    2 |   sellSideMM |
      | trader1 |   140 |   11 |   sellSideMM |
      
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |     56 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   3588 |    6468 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         897 |   2870 |    3588 |    4485 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |     13 |            56 |           0 |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancels the following orders reference:
      | trader    | reference |
      | buySideMM |      buy1 |
      | buySideMM |      buy2 |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   3588 |    6468 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         897 |   2870 |    3588 |    4485 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |     13 |            56 |           0 |
    
    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 100
    Then traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell |      1 |   100 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   100 |      1 | LIMIT | GTC |

    And the following transfers happened:
      | from    | to      | fromType   |     toType | id        | amount | asset |
      | trader1 | market  | MARGIN     | SETTLEMENT | ETH/DEC19 |    520 | ETH   |
    
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   1560 |    7976 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         390 |   1248 |    1560 |    1950 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |     13 |          -464 |           0 |
    
    # PARTIAL CLOSEOUT BY TRADER
    Then traders place following orders:
      | trader  | market id | type | volume | price | trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     10 |    80 |      1 | LIMIT | GTC |
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |     96 |    9180 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |          24 |     76 |      96 |     120 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |      3 |          -167 |        -557 |

  # FULL CLOSEOUT BY TRADER
  Then traders place following orders:
    | trader  | market id | type | volume | price | trades | type  | tif |
    | trader1 | ETH/DEC19 | sell |      3 |    70 |      1 | LIMIT | GTC |
  And I expect the trader to have a margin:
    | trader  | asset | market id | margin | general |
    | trader1 | ETH   | ETH/DEC19 |      0 |    9276 |
  And the margins levels for the traders are:
    | trader  | market id | maintenance | search | initial | release |
    | trader1 | ETH/DEC19 |           0 |      0 |       0 |       0 |
  And position API produce the following:
    | trader  | volume | unrealisedPNL | realisedPNL |
    | trader1 |      0 |             0 |        -724 |