Feature: Trader maxes out their margin account but isn't liquidated. They try to submit an order that will reduce their position.
  # https://github.com/vegaprotocol/product/issues/223

  # we have 5 decimal places so 94.0 is 9400000. 
  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | tau/long | lamd/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | BTC      | ETH       | ETH   | 9400000   | simple     | 0.2      | 0.2        | 0  | 0 | 0     | 5              | 3              | 2             | 9400000         | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |
    And the following traders:
      | name       | amount     |
      | trader1    | 245599970  |
      | sellSideMM | 1000000000 |
      | buySideMM  | 1000000000 |
    # setting mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10300000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1      | TYPE_LIMIT | TIF_GTC |


    # setting up order book
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
      | trader1 | ETH/DEC19 | sell | 13     | 0       | 3      | TYPE_MARKET | TIF_IOC |

    And executed trades:
      | buyer     | price    | size | seller  |
      | buySideMM | 10000000 | 1    | trader1 |
      | buySideMM | 9600000  | 3    | trader1 |
      | buySideMM | 9000000  | 9    | trader1 |

    And the mark price for the market "ETH/DEC19" is "9000000"

    #And "trader1" general account for asset "ETH" balance is "0"

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 248399970 | 0         |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 82799990    | 165599980 | 248399970 | 413999950 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -13    | 2800000       | 0           |

    #
    # reducing position when not maxed out with a market order 
    # 
    Then traders place following orders:
      | trader  | market id | type | volume | price   | trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  | 2      | 0       | 1      | TYPE_MARKET | TIF_IOC |
    
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 219799970 | 0         |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 55440000    | 110880000 | 166320000 | 277200000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -11    | -21830769     | -3969231    |

    And the mark price for the market "ETH/DEC19" is "11200000"

    # setting new mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 14000000 | 1      | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "14000000"

    #And I expect the trader to have a margin:
    #  | trader  | asset | market id | margin    | general   |
    #  | trader1 | ETH   | ETH/DEC19 | 248399970 | 0         |

    #And the margins levels for the traders are:
    #  | trader  | market id | maintenance | search    | initial   | release   |
    #  | trader1 | ETH/DEC19 | 55440000    | 110880000 | 166320000 | 277200000 |
    


















    # Then the following transfers happened:
    #   | from   | to      | fromType                | toType              | id        | amount  | asset |
    #  | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 248399970 | ETH   |
