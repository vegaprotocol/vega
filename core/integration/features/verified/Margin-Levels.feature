Feature: Check the margin scaling levels (maintenance, search, initial, release) are correctly applied to the maintenance margin that is calculated by the risk model

  Background:
    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the margin calculator named "margin-calculator-0":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring | data source config          |
      | ETH/DEC19 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-0 | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: 0010-MARG-015,0010-MARG-016,0010-MARG-017
    Given the parties deposit on asset's general account the following amount:
      | party       | asset | amount        |
      | auxiliary1  | USD   | 1000000000000 |
      | auxiliary2  | USD   | 1000000000000 |
      | auxiliary10 | USD   | 1000000000000 |
      | auxiliary20 | USD   | 1000000000000 |
      | trader2     | USD   | 10000         |
      | trader3     | USD   | 9000          |
      | trader20    | USD   | 10000         |
      | trader30    | USD   | 9000          |
      | lprov       | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendmend  |
      | lp0 | lprov | ETH/DEC20 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp0 | lprov | ETH/DEC20 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendmend  |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    # trading happens at the end of the open auction period
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | auxiliary2 | ETH/DEC19 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5     |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000  |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1     |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1     |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-50    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10000 |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-10    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10    |

    When the opening auction period ends for market "ETH/DEC19"
    When the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "10" for the market "ETH/DEC19"
    And the mark price should be "10" for the market "ETH/DEC20"

    # setup trader2 position to be ready to takeover trader3's position once trader3 is closed out
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader2  | ETH/DEC19 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
      | trader20 | ETH/DEC20 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |

    And the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | trader2  | ETH/DEC19 | 321         | 481    | 642     | 963     |
      | trader20 | ETH/DEC20 | 321         | 385    | 481     | 642     |

    # margin level = OrderSize*MarkPrice*RF = 40*10*0.801225765=321

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader2  | USD   | ETH/DEC19 | 642    | 9358    |
      | trader20 | USD   | ETH/DEC20 | 481    | 9519    |

    When the parties place the following orders with ticks:
      | party       | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | auxiliary1  | ETH/DEC19 | sell | 40     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
      | auxiliary10 | ETH/DEC20 | sell | 40     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-3 |

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | trader2  | 40     | 0              | 0            |
      | trader20 | 40     | 0              | 0            |

    # check margin initial level
    # trader2 and trader20 have open position of 40 now
    And the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | trader2  | ETH/DEC19 | 3402        | 5103   | 6804    | 10206   |
      | trader20 | ETH/DEC20 | 3402        | 4082   | 5103    | 6804    |
      #| trader2  | ETH/DEC19 | 1602        | 2403   | 3204    | 4806    |
      #| trader20 | ETH/DEC20 | 1602        | 1922   | 2403    | 3204    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader2  | USD   | ETH/DEC19 | 6804   | 3196    |
      #| trader2  | USD   | ETH/DEC19 | 3204   | 5796    |
      | trader20 | USD   | ETH/DEC20 | 5103   | 4897    |
      #| trader20 | USD   | ETH/DEC20 | 2403   | 6597    |

    # move mark price from 50 to 20, MTM, hence cash flow beween margin and general account for trader2 and trader20
    When the parties place the following orders with ticks:
      | party       | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | auxiliary1  | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-4  |
      | auxiliary10 | ETH/DEC19 | sell | 1      | 20    | 1                | TYPE_LIMIT | TIF_GTC | sell-order-4 |
      | auxiliary1  | ETH/DEC20 | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-5  |
      | auxiliary10 | ETH/DEC20 | sell | 1      | 20    | 1                | TYPE_LIMIT | TIF_GTC | sell-order-5 |

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | trader2  | 40     | -1200          | 0            |
      | trader20 | 40     | -1200          | 0            |

    And the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | trader2  | ETH/DEC19 | 1241        | 1861   | 2482    | 3723    |
      #| trader2  | ETH/DEC19 | 641         | 961    | 1282    | 1923    |
      | trader20 | ETH/DEC20 | 1241        | 1489   | 1861    | 2482    |
      #| trader20 | ETH/DEC20 | 641         | 769    | 961     | 1282    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader2  | USD   | ETH/DEC19 | 2482   | 6318    |
      #| trader2  | USD   | ETH/DEC19 | 1282   | 6518    |
      | trader20 | USD   | ETH/DEC20 | 1861   | 6939    |
      #| trader20 | USD   | ETH/DEC20 | 1203   | 6597    |

    # check margin release level
    # MTM process will reduce (50-20)*40=1200 from general account
    # for trader2: MTM brings margin account from 3204 to 2204 which is above release level, so margin account has been set to initial level: 1282
    # for trader 20: MTM brings margin account from 2403 to 1203 which is below release level, so margin account is kept at 1203

    When the parties place the following orders with ticks:
      | party       | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | trader2     | ETH/DEC19 | sell | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sell-order-6 |
      | trader20    | ETH/DEC20 | sell | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sell-order-6 |
      | auxiliary1  | ETH/DEC19 | buy  | 40     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-6  |
      | auxiliary10 | ETH/DEC20 | buy  | 40     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-6  |

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | trader2  | 0      | 0              | 0            |
      | trader20 | 0      | 0              | 0            |

    And the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | trader2  | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | trader20 | ETH/DEC20 | 0           | 0      | 0       | 0       |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader2  | USD   | ETH/DEC19 | 0      | 10000   |
      | trader20 | USD   | ETH/DEC20 | 0      | 10000   |

    When the parties place the following orders with ticks:
      | party       | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | trader2     | ETH/DEC19 | sell | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sell-order-6 |
      | trader20    | ETH/DEC20 | sell | 20     | 50    | 0                | TYPE_LIMIT | TIF_GTC | sell-order-6 |
      | auxiliary1  | ETH/DEC19 | buy  | 20     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-6  |
      | auxiliary10 | ETH/DEC20 | buy  | 20     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-6  |

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | trader2  | 0      | 0              | -10000       |
      #| trader2  | -20    | 0              | 0            |
      | trader20 | 0      | 0              | -10000       |
      #| trader20 | -20    | 0              | 0            |

      #And the parties should have the following margin levels:
      #| party    | market id | maintenance | search | initial | release |
      #| trader2  | ETH/DEC19 | 4657        | 6985   | 9314    | 13971   |
      #| trader20 | ETH/DEC20 | 4657        | 5588   | 6985    | 9314    |

    # With the different MTM approach, these traders now get closed out
    # check margin search level
    #mentainance level before new open position: margin_trader2 = 20*50*3.55690359157934000=3557
    #initial level: margin_trader2 = 20*50*3.55690359157934000*2=7114 which is more than search level, so margin account is set at 7114
    #mentainance level before new open position: margin_trader20 = 20*50*3.55690359157934000=3557
    #initial level: margin_trader20 = 20*50*3.55690359157934000*1.5=5336 which is less than search level and higher than maintenance

    #Then the parties should have the following account balances:
    #  | party    | asset | market id | margin | general |
    #  | trader2  | USD   | ETH/DEC19 | 7114   | 1886    |
    #  | trader20 | USD   | ETH/DEC20 | 6985   | 2015    |


