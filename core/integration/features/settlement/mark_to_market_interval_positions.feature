Feature: Check position tracking matches expected behaviour with MTM intervals. Based on position_tracking/verified-positions-resolution-5-lognormal

  Background:
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    #calculated risk factor long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999999  | 300               |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring | data source config     |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 5s    |

  Scenario: using lognormal risk model, set "designatedLoser" closeout while the position of "designatedLoser" is not fully covered by orders on the order book (0007-POSN-013)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | USD   | 1000000000000 |
      | buySideProvider  | USD   | 1000000000000 |
      | designatedLoser  | USD   | 21600         |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lpprov           | USD   | 1000000000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | amendment  |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond  |
      | lpprov | USD   | ETH/DEC19 | 0      | 999999910000 | 90000 |

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
      | sell | 2100  | 0      |
      | sell | 2000  | 10     |
      | buy  | 1     | 90010  |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general      | bond  |
      | lpprov | USD   | ETH/DEC19 | 6821442 | 999993088558 | 90000 |

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
      | party  | asset | market id | margin | general      | bond  |
      | lpprov | USD   | ETH/DEC19 | 170536 | 999999739464 | 90000 |

    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC19 | 17250  | 0       |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2100  | 43     |
      | sell | 2000  | 10     |
      | buy  | 1     | 10     |
      | buy  | 40    | 2250   |
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
      | designatedLoser | ETH/DEC19 | 0           | 0      | 0       | 0       |

