Feature: Check position tracking matches expected behaviour with MTM intervals. Based on position_tracking/verified-positions-resolution-5-lognormal

  Background:
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    #long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.99999999  | 300               |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none       | default-eth-for-future | 1e0                    | 0                         |
      | ETH/DEC20 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 5s    |

  Scenario: 001, using lognormal risk model, set "designatedLoser" closeout while the position of "designatedLoser" is not fully covered by orders on the order book (0007-POSN-013)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | USD   | 1000000000000 |
      | buySideProvider  | USD   | 1000000000000 |
      | designatedLoser  | USD   | 21981         |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lpprov           | USD   | 1000000000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | sell | ASK              | 50         | 100    | amendment  |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | lpprov | USD   | ETH/DEC19 | 0      | 999999991000 | 9000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2001  | 5      |
      | sell | 2000  | 10     |
      | buy  | 1     | 9010   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | lpprov | USD   | ETH/DEC19 | 682144 | 999999308856 | 9000 |

    # insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    # insurance pool generation - trade
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | lpprov | USD   | ETH/DEC19 | 17055  | 999999973945 | 9000 |

    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC19 | 17631  | 0       |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2100  | 5      |
      | sell | 2000  | 10     |
      | buy  | 1     | 10     |
      | buy  | 40    | 225    |
      | buy  | 140   | 1      |

    #designatedLoser has position of vol 290; price 150; calculated risk factor long: 0.336895684; risk factor short: 0.4878731
    #what's on the order book to cover the position is shown above, which makes the exit price 38.65517241 =(1*10+40*280)/290, slippage per unit is 150-38.65517241=111.345
    #margin level is PositionVol*(markPrice*RiskFactor+SlippagePerUnit) = 290*(150*0.336895684+111.345)=46946

    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC19 | 14654       | 17584  | 21981   | 29308   |

    # Moving time forward 1 block, should trigger MTM
    When the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC19 | 14654       | 17584  | 21981   | 29308   |

    # Add another 4 blocks, and we will have crossed over the threshold, and we will MTM
    When the network moves ahead "4" blocks
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC19 | 58154       | 69784  | 87231   | 116308  |

    # insurance pool generation - modify order book
    And the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 290    | 20    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    # insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the network moves ahead "6" blocks

    Then the following trades should be executed:
      | buyer           | price | size | seller           |
      | buySideProvider | 140   | 1    | sellSideProvider |
      | buySideProvider | 20    | 290  | network          |
      | network         | 20    | 290  | designatedLoser  |

    Then the following network trades should be executed:
      | party           | aggressor side | volume |
      | buySideProvider | sell           | 290    |
      | designatedLoser | buy            | 290    |

    # check positions and verify loss socialisation is reflected in realised P&L (0007-POSN-013)
    Then the parties should have the following profit and loss:
      | party           | volume | unrealised pnl | realised pnl |
      | designatedLoser | 0      | 0              | -17631       |
      | buySideProvider | 291    | 34800          | -20649       |

    # check margin levels
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC19 | 0           | 0      | 0       | 0       |
    # checking margins
    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds (however they get later spent on MTM payment to closeout-facilitating party)
    Then the following transfers should happen:
      | from            | to              | from account            | to account                       | market id | amount | asset |
      | designatedLoser | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 0      | USD   |
      | buySideProvider | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 | 14     | USD   |
      | designatedLoser |                 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC19 | 0      | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 | 0      | USD   |
      | designatedLoser | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_INSURANCE           | ETH/DEC19 | 14151  | USD   |
      | market          | market          | ACCOUNT_TYPE_INSURANCE  | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC19 | 14151  | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 14151  | USD   |
      | buySideProvider | buySideProvider | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 76     | USD   |
      | buySideProvider | buySideProvider | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 21981  | USD   |
      | buySideProvider | buySideProvider | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 76     | USD   |
      | buySideProvider | buySideProvider | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 45052  | USD   |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"


  Scenario: 002, closeout trade with price outside price mornitoring bounds will not trigger auction 0032-PRIM-019
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | USD   | 1000000000000 |
      | buySideProvider  | USD   | 1000000000000 |
      | designatedLoser  | USD   | 21981         |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lpprov           | USD   | 1000000000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 9000              | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 9000              | 0.1 | sell | ASK              | 50         | 100    | amendment  |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | lpprov | USD   | ETH/DEC20 | 0      | 999999991000 | 9000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "150" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | sell | 2001  | 5      |
      | sell | 2000  | 10     |
      | buy  | 1     | 9010   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | lpprov | USD   | ETH/DEC20 | 682144 | 999999308856 | 9000 |

    # insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC20 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC20 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    # insurance pool generation - trade
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser | ETH/DEC20 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | lpprov | USD   | ETH/DEC20 | 17055  | 999999973945 | 9000 |

    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC20 | 17631  | 0       |

    Then the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | sell | 2100  | 5      |
      | sell | 2000  | 10     |
      | buy  | 1     | 10     |
      | buy  | 40    | 225    |
      | buy  | 140   | 1      |

    #designatedLoser has position of vol 290; price 150; calculated risk factor long: 0.336895684; risk factor short: 0.4878731
    #what's on the order book to cover the position is shown above, which makes the exit price 38.65517241 =(1*10+40*280)/290, slippage per unit is 150-38.65517241=111.345
    #margin level is PositionVol*(markPrice*RiskFactor+SlippagePerUnit) = 290*(150*0.336895684+111.345)=46946

    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC20 | 14654       | 17584  | 21981   | 29308   |

    # Moving time forward 1 block, should trigger MTM
    When the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC20 | 14654       | 17584  | 21981   | 29308   |

    # Add another 4 blocks, and we will have crossed over the threshold, and we will MTM
    When the network moves ahead "4" blocks
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC20 | 58154       | 69784  | 87231   | 116308  |

    # insurance pool generation - modify order book
    And the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC20 | buy  | 290    | 20    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    # insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC20 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC20 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the network moves ahead "6" blocks

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 140        | TRADING_MODE_CONTINUOUS | 3600    | 140       | 161       | 397516       | 9000           | 292           | 1                     | 1000             | 2000                    |

    Then the following trades should be executed:
      | buyer           | price | size | seller           |
      | buySideProvider | 140   | 1    | sellSideProvider |
      | buySideProvider | 20    | 290  | network          |
      | network         | 20    | 290  | designatedLoser  |
    # closeout trade price is 20 which is outside price mornitoring bounds, and does not trigger auction
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the following network trades should be executed:
      | party           | aggressor side | volume |
      | buySideProvider | sell           | 290    |
      | designatedLoser | buy            | 290    |

    Then the parties should have the following profit and loss:
      | party           | volume | unrealised pnl | realised pnl |
      | designatedLoser | 0      | 0              | -17631       |
      | buySideProvider | 291    | 34800          | -20649       |

    # check margin levels
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC20 | 0           | 0      | 0       | 0       |
    # checking margins
    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC20 | 0      | 0       |

  Scenario: 003, settlement works correctly when party enters and leaves within one MTM window
    Given the markets are updated:
      | id        | linear slippage factor | quadratic slippage factor | lp price range |
      | ETH/DEC19 | 1e0                    | 0                         | 2              |
    And the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lp               | USD   | 1000000000000 |
      | buyer            | USD   |       1000000 |
      | seller           | USD   |       1000000 |
      | party1           | USD   |       1000000 |
      | party2           | USD   |       1000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 10s   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp     | ETH/DEC19 | 9000              | 0.0 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lp     | ETH/DEC19 | 9000              | 0.0 | sell | ASK              | 50         | 100    | amendment  |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            |
      | 150        | TRADING_MODE_CONTINUOUS |

    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer  | ETH/DEC19 | buy  |     6 |    90 | 0                | TYPE_LIMIT | TIF_GTC |
 
    And the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer  | ETH/DEC19 | buy  |     9  |    95 | 0                | TYPE_LIMIT | TIF_GTC |
    
    And the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer  | ETH/DEC19 | buy  |      5 |   100 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell |     11 |    90 | 2                | TYPE_LIMIT | TIF_FOK |
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            |
      | 150        | TRADING_MODE_CONTINUOUS |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | buyer  |   100 |    5 | party1 | 
      | buyer  |    95 |    6 | party1 | 
     
    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell |      9 |    87 | 2                | TYPE_LIMIT | TIF_FOK|
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | buyer  |    95 |    3 | party2 | 
      | buyer  |    90 |    6 | party2 | 
    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party1 | ETH/DEC19 |         805 |
      | party2 | ETH/DEC19 |         659 |

    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | seller | ETH/DEC19 | sell |     8  |    90 | 0                | TYPE_LIMIT | TIF_GTC |
      | seller | ETH/DEC19 | sell |     7  |    85 | 0                | TYPE_LIMIT | TIF_GTC |
      | seller | ETH/DEC19 | sell |     6  |    80 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | buy  |     11 |    90 | 2                | TYPE_LIMIT | TIF_FOK |
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 |    80 |    6 | seller | 
      | party1 |    85 |    5 | seller | 
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            |
      | 150        | TRADING_MODE_CONTINUOUS |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 |      0 | 1000000 |
      | party2 | USD   | ETH/DEC19 |    988 |  999012 |
  
    # Go to next MTM window
    When the network moves ahead "5" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 |      0 | 1000165 |
      | party2 | USD   | ETH/DEC19 |    601 |  999459 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 |  0     |              0 |          165 |
      | party2 | -9     |             60 |            0 |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            |
      | 85        | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  |      9 |    90 | 2                | TYPE_LIMIT | TIF_FOK |
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 |    85 |    2 | seller | 
      | party2 |    90 |    7 | seller | 

    # Go to next MTM window
    When the network moves ahead "9" blocks
    # party1 gains: 100*5+95*6-80*6-85*5=165
    # party2 gains:  95*3+90*6-85*2-90*7=25
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 |      0 | 1000165 |
      | party2 | USD   | ETH/DEC19 |      0 | 1000025 |
      | buyer  | USD   | ETH/DEC19 |   3479 |  996426 |
      | seller | USD   | ETH/DEC19 |   2674 |  997231 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 |      0 |              0 |          165 |
      | party2 |      0 |              0 |           25 |
      | buyer  |     20 |            -95 |            0 |
      | seller |    -20 |            -95 |            0 |