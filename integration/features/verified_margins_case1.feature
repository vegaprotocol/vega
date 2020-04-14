Feature: CASE-1: Trader submits long order that will trade - new formula & high exit price
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
      | buySideMM  | ETH/DEC19 |  buy |     15 |    90 |      0 | LIMIT | GTC |      buy3 |
      | buySideMM  | ETH/DEC19 |  buy |     50 |    87 |      0 | LIMIT | GTC |     _buy4 |


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
    And "trader1" general account for asset "ETH" balance is "6832"
    And executed trades:
      |  buyer  | price | size |       seller |
      | trader1 |   112 |    2 |   sellSideMM |
      | trader1 |   140 |   11 |   sellSideMM |
      
    Then the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |     56 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   3224 |    6832 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         806 |   2579 |    3224 |    4030 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |     13 |            56 |           0 |

    # NEW ORDERS ADDED WITHOUT ANOTHER TRADE HAPPENING
    Then traders cancels the following orders reference:
      | trader    | reference |
      | buySideMM |      buy1 |
      | buySideMM |      buy2 |
      | buySideMM |      buy3 |
    Then traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   190 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      3 |   180 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |     15 |   170 |      0 | LIMIT | GTC |
    
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   3224 |    6832 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         806 |   2579 |    3224 |    4030 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |     13 |            56 |           0 |
    
    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 200
    Then traders place following orders:
      | trader     | market id | type | volume | price | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell |      1 |   200 |      0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   200 |      1 | LIMIT | GTC |
    
    And the following transfers happened:
      | from   | to      | fromType   | toType | id        | amount | asset |
      | market | trader1 | SETTLEMENT | MARGIN | ETH/DEC19 |    780 | ETH   |
    
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   2444 |    8392 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search | initial | release |
      | trader1 | ETH/DEC19 |         611 |   1955 |    2444 |    3055 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |     13 |           836 |           0 |
    
    # FULL CLOSEOUT BY TRADER
    Then traders place following orders:
      | trader  | market id | type | volume | price | trades | type  | tif |
      | trader1 | ETH/DEC19 | sell |     13 |   165 |      3 | LIMIT | GTC |
  And the margins levels for the traders are:
    | trader  | market id | maintenance | search | initial | release |
    | trader1 | ETH/DEC19 |           0 |      0 |       0 |       0 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 |      0 |             0 |         496 |