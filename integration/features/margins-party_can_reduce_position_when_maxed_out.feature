Feature: Trader maxes out their margin account but isn't liquidated. They try to submit an order that will reduce their position.
  # https://github.com/vegaprotocol/product/issues/223

  # we have 5 decimal places so 94.0 is 9400000. 
  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | tau/long | lamd/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | BTC      | ETH       | ETH   | 9400000   | simple     | 0.2      | 0.2        | 0  | 0 | 0     | 100            | 1.1            | 1.05          | 9400000         | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |
    And the following traders:
      | name       | amount     |
      | trader1    | 2398000    |
      | sellSideMM | 1000000000 |
      | buySideMM  | 1000000000 |
    # setting mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10000000 | 0      | TYPE_LIMIT | TIF_GTC |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10000000 | 1      | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "10000000"

    # setting up order book, note that 'sell4' price is crucial so that slippage maxes the trader out
    And traders place following orders with references:
      | trader     | market id | type | volume | price    | trades | type  | tif | reference |
      | sellSideMM | ETH/DEC19 | sell | 1      | 13840000 | 0      | TYPE_LIMIT | TIF_GTC | sell4     |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10250000 | 0      | TYPE_LIMIT | TIF_GTC | sell3     |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10200000 | 0      | TYPE_LIMIT | TIF_GTC | sell2     |
      | sellSideMM | ETH/DEC19 | sell | 1      | 10100000 | 0      | TYPE_LIMIT | TIF_GTC | sell1     |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 9900000  | 0      | TYPE_LIMIT | TIF_GTC | buy1      |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 9800000  | 0      | TYPE_LIMIT | TIF_GTC | buy2      |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 9700000  | 0      | TYPE_LIMIT | TIF_GTC | buy3      |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 9600000  | 0      | TYPE_LIMIT | TIF_GTC | buy4      |


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
      | trader1 | ETH/DEC19 | sell | 1      | 0       | 1      | TYPE_MARKET | TIF_IOC |

    And executed trades:
      | buyer     | price    | size | seller  |
      | buySideMM | 9900000  | 1    | trader1 |
      
    And the mark price for the market "ETH/DEC19" is "9900000"

    #And "trader1" general account for asset "ETH" balance is "0"

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 2398000   | 0         |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 2180000     | 2289000   | 2398000   | 218000000 |
    And position API produce the following:
      | trader  | volume | unrealisedPNL | realisedPNL |
      | trader1 | -1     | 0             | 0           |
    

    # setting new mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10100000 | 1      | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "10100000"

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 2198000   | 0         |
    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 2120000     | 2226000   | 2332000   | 212000000 |
    

    # setting new mark price
    And traders place following orders:
      | trader     | market id | type | volume | price    | trades | type  | tif |
      | buySideMM  | ETH/DEC19 | buy  | 1      | 10200000 | 1      | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC19" is "10200000"

    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 2098000   | 0         |

    And the margins levels for the traders are:
      | trader  | market id | maintenance | search    | initial   | release   |
      | trader1 | ETH/DEC19 | 2090000     | 2194500   | 2299000   | 209000000 |
    
    # the difference between maintenance and margin is 0.8; the price is 102.0 and the next sell is at 102.5 
    # so the P&L should be 0.5 which is within 0.8 ... so it's better to close out now and come out with 0.3 
    # trader1 tries to close out
    Then traders place following orders:
      | trader  | market id | type | volume | price   | trades | type  | tif |
      | trader1 | ETH/DEC19 | buy  | 1      | 0       | 1      | TYPE_MARKET | TIF_IOC |

    And the mark price for the market "ETH/DEC19" is "10250000"
    And I expect the trader to have a margin:
      | trader  | asset | market id | margin    | general   |
      | trader1 | ETH   | ETH/DEC19 | 0         | 2048000   |


    # setting new mark price
    # And traders place following orders:
    #   | trader     | market id | type | volume | price    | trades | type  | tif |
    #   | buySideMM  | ETH/DEC19 | buy  | 1      | 10300000 | 1      | TYPE_LIMIT | TIF_GTC |

    # And the mark price for the market "ETH/DEC19" is "10300000"


    # And I expect the trader to have a margin:
    #   | trader  | asset | market id | margin    | general   |
    #   | trader1 | ETH   | ETH/DEC19 | 5600000   | 0         |

    # And the margins levels for the traders are:
    #   | trader  | market id | maintenance | search    | initial    | release   |
    #   | trader1 | ETH/DEC19 | 5600000     | 11200000  | 16800000   | 28000000  |
    
    # We see the trader is exactly maxed out - maintenance level is their margin balance

















    # Then the following transfers happened:
    #   | from   | to      | fromType                | toType              | id        | amount  | asset |
    #  | market | trader1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 248399970 | ETH   |
