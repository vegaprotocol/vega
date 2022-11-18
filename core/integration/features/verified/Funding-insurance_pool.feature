Feature: Position resolution case 5 lognormal risk model

    Background:

    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r     | sigma |
      | 0.001         | 0.01 | 0  | 0.0   | 1.2   |
      #calculated risk factor long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999999  | 300               |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring  | data source config          |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none      | default-eth-for-future |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    Scenario: using lognormal risk model, set "designatedLooser" closeout while the position of "designatedLooser" is not fully covered by orders on the order book; 0012-POSR-002, 0012-POSR-005

# setup accounts
     Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | USD   | 1000000000000 |
      | buySideProvider  | USD   | 1000000000000 |
      | designatedLooser | USD   | 21600         |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lpprov           | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | amendment |

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

# insurance pool generation - setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | 731          | 0              |     1         |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf=150*10*1*0.4878731=731

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | volume   | price  |
      | buy  | 10       | 1      |
      | sell | 10       | 2000   |

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |

# insurance pool generation - trade
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLooser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party            | asset | market id | margin  | general |
      | designatedLooser | USD   | ETH/DEC19 | 21600   | 0       |

    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search  | initial | release |
      | designatedLooser | ETH/DEC19 | 54384       | 65260   | 81576   | 108768   |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price  | volume |
      | buy  | 1      | 10     |
      | buy  | 140    | 1      |

  #designatedLooser has position of vol 290; price 150; calculated risk factor long: 0.336895684; risk factor short: 0.4878731
  #what's on the order book to cover the position is shown above, which makes the exit price 13 =(1*10+140*1)/11, slippage per unit is 150-13=137
  #margin level is PositionVol*(markPrice*RiskFactor+SlippagePerUnit) = 290*(150*0.336895684+137)=54384

# insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    When the parties place the following orders with ticks:
      | party           | market id | side | volume   | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 290      | 120    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

# insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the following trades should be executed:
      | buyer           | price | size | seller           |
      | buySideProvider |   140 | 1    | sellSideProvider |
      | buySideProvider |   120 | 290  | network          |
      | network         |   120 | 290  | designatedLooser |

    Then the following network trades should be executed:
      | party            | aggressor side | volume |
      | buySideProvider  | sell           | 290    |
      | designatedLooser | buy            | 290    |

    # check positions
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -21600       |
      | sellSideProvider | -291   | 2900           | 0            |
      | buySideProvider  |  291   | 5800           | 0            |
      | aux              | 1      | -10            | 0            |
      | aux2             | -1     | 10             | 0            |
      | lpprov           | 0      | 0              | 0            |

    Then the parties should have the following account balances:
      | party            | asset | market id | margin  | general            |
      | designatedLooser | USD   | ETH/DEC19 | 0       | 0                  |
      | sellSideProvider | USD   | ETH/DEC19 | 839594  | 999999163306       |
      | buySideProvider  | USD   | ETH/DEC19 | 81259   | 999999924541       |
      | aux              | USD   | ETH/DEC19 | 1088    | 999999998902       |
      | aux2             | USD   | ETH/DEC19 | 2896    | 999999997114       |

# check margin levels
    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search  | initial  | release |
      | designatedLooser | ETH/DEC19 | 0           | 0       | 0        | 0       |
# checking margins
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLooser | USD   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds (however they get later spent on MTM payment to closeout-facilitating party)
    Then the following transfers should happen:
      | from             | to              | from account            | to account                       | market id | amount | asset |
      | designatedLooser | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 |      0 |   USD |
      | designatedLooser | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 |      0 |   USD |
      | designatedLooser |                 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC19 |      0 |   USD |
      | market           | buySideProvider | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 |      0 |   USD |
      | designatedLooser | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_INSURANCE           | ETH/DEC19 |  18700 |   USD |
      | market           | market          | ACCOUNT_TYPE_INSURANCE  | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC19 |   5800 |   USD |
      | market           | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 |   5800 |   USD |

    And the insurance pool balance should be "12900" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 120        | TRADING_MODE_CONTINUOUS | 340728       | 0              |     291       |

    Then the following transfers should happen:
      | from            | to              | from account            | to account              | market id | amount | asset |
      | buySideProvider | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 5820   | USD   |

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLooser | 0      | 0              | -21600       |
      | sellSideProvider | -291   | 8720           | 0            |
      | buySideProvider  | 291    | -20            | 0            |
      | aux              | 0      | 0              | -30          |
      | aux2             | 0      | 0              | 30           |
      | lpprov           | 0      | 0              | 0            |

    # MTM: buySideProvider's unrealised pnl has been updated from 34800 to 28980, which is 291*(140-120)=5820=34800-28980,
    # MTM: buySideProvider's margin account has been updated from 81259 to 75439, which is 291*(140-120)=5820

     Then the parties should have the following account balances:
      | party            | asset | market id | margin  | general            |
      | designatedLooser | USD   | ETH/DEC19 | 0       | 0                  |
      | sellSideProvider | USD   | ETH/DEC19 | 845414  | 999999163306       |
      | buySideProvider  | USD   | ETH/DEC19 | 75439   | 999999924541       |
      | aux              | USD   | ETH/DEC19 | 1108    | 999999998862       |
      | aux2             | USD   | ETH/DEC19 | 0       | 1000000000030      |
