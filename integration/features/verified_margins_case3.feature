Feature: CASE-3: Trader submits long order that will trade - new formula & zero side of order book
  # https://drive.google.com/drive/folders/1BCOKaEb7LZYAKoiPfXfaqwM4BNicPpF-

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | tau/short | lamd/long | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | ETH        | ETH   | 9400000    | simple     | 0.1       | 0.2       | 0              | 0               | 0     | 5              | 4              | 3.2           | 9400000          | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |
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
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 100    | 25000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 11     | 14000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | sellSideMM | ETH/DEC19 | sell | 2      | 11200000 | 0      | TYPE_LIMIT | TIF_GTC |


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
    And "trader1" general account for asset "ETH" balance is "860000000"
    And executed trades:
      | buyer   | price    | size | seller     |
      | trader1 | 11200000 | 2    | sellSideMM |
      | trader1 | 14000000 | 11   | sellSideMM |

    Then the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount  | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 5600000 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin   | general   |
      | trader1 | ETH   | ETH/DEC19 | 145600000 | 860000000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search   | initial  | release  |
      | trader1 | ETH/DEC19 | 36400000    | 116480000 | 145600000 | 182000000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 13     | 5600000       | 0           |

    # ANOTHER TRADE HAPPENING (BY A DIFFERENT PARTY)
    # updating mark price to 160
    Then traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 1      | 16000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 16000000 | 1      | TYPE_LIMIT | TIF_GTC |

    And the following transfers happened:
      | from   | to      | fromType                | toType              | id        | amount   | asset |
      | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 26000000 | ETH   |

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin   | general   |
      | trader1 | ETH   | ETH/DEC19 | 171600000 | 860000000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search   | initial  | release   |
      | trader1 | ETH/DEC19 | 41600000    | 133120000 | 166400000 | 208000000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 13     | 31600000      | 0           |

    # CLOSEOUT ATTEMPT (FAILED, no buy-side in order book) BY TRADER
    Then traders place following orders:
      | trader  | market id | type | volume | price   | trades | type  | tif |
      | trader1 | ETH/DEC19 | sell | 13     | 8000000 | 0      | TYPE_LIMIT | TIF_GTC |
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin   | general   |
      | trader1 | ETH   | ETH/DEC19 | 171600000 | 860000000 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search   | initial  | release   |
      | trader1 | ETH/DEC19 | 41600000    | 133120000 | 166400000 | 208000000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | 13     | 31600000      | 0           |
