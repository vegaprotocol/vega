Feature: Closeout scenarios
  # This is a test case to demonstrate an order can be rejected when the trader (who places an initial order) does not have enouge collateral to cover the initial margin level

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring | data source config     |
      | ETH/DEC19 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | default-basic    | default-eth-for-future |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: 2 parties get close-out at the same time. Distressed position gets taken over by LP, distressed order gets canceled (0005-COLL-002; 0012-POSR-001; 0012-POSR-002; 0012-POSR-004; 0012-POSR-005)
    # setup accounts, we are trying to closeout trader3 first and then trader2

    Given the insurance pool balance should be "0" for the market "ETH/DEC19"

    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 2000          |
      | trader3    | USD   | 90            |
      | lprov      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp1 | lprov | ETH/DEC19 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendmend  |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    # trading happens at the end of the open auction period
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC19 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC19 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC19 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the auction ends with a traded volume of "10" at a price of "10"
    And the mark price should be "10" for the market "ETH/DEC19"

    # setup trader2 position to be ready to takeover trader3's position once trader3 is closed out
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader2 | ETH/DEC19 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 50    | 4040   |
      | sell | 1000  | 10     |
      | sell | 1055  | 223    |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 321         | 481    | 642     | 963     |
      | lprov   | ETH/DEC19 | 32030       | 48045  | 64060   | 96090   |
    # margin level_trader2= OrderSize*MarkPrice*RF = 40*10*0.801225765=321
    # margin level_Lprov= OrderSize*MarkPrice*RF = max(223*10*3.55690359157934000,4040*10*0.801225765)=32370


    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 642    | 1358    |

    # setup trader3 position and close it out
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | trader3 | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | buy-position-31 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 5      |
      | buy  | 45    | 278966 |
      | buy  | 50    | 40     |
      | buy  | 100   | 10     |
      | sell | 1000  | 10     |
      | sell | 1055  | 223    |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC19 | 81          | 121    | 162     | 243     |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 642    | 1358    |
      | trader3 | USD   | ETH/DEC19 | 90     | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party      | volume | unrealised pnl | realised pnl |
      | auxiliary1 | -10    | 0              | 0            |
      | auxiliary2 | 10     | 0              | 0            |
    #setup trader3 position and close it out
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | auxiliary2 | ETH/DEC19 | sell | 10     | 100   | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 40005  |
      | buy  | 45    | 0      |
      | buy  | 50    | 0      |
      | buy  | 100   | 0      |
      | sell | 1000  | 10     |
      | sell | 1055  | 223    |
    #trader3 is closed out
    #trader2's order is canceled since mark price has moved from 10 to 100, hence margin level has increased by 10 times
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | trader3 | ETH/DEC19 | 0           | 0      | 0       | 0       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC19 | 0      | 2000    |
      | trader3 | USD   | ETH/DEC19 | 0      | 0       |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party      | volume | unrealised pnl | realised pnl |
      | auxiliary1 | -10    | -900           | 0            |
      | auxiliary2 | 0      | 0              | 900          |
      | trader2    | 0      | 0              | 0            |
      | trader3    | 0      | 0              | -90          |
      | lprov      | 10     | 500            | -411         |
      #| lprov      | 10     | 550            | -461         |
    And the mark price should be "100" for the market "ETH/DEC19"

Scenario: Position becomes distressed upon exiting an auction (0012-POSR-007)
    Given the insurance pool balance should be "0" for the market "ETH/DEC19"
    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 1027          |
      | lprov      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov | ETH/DEC20 | 100000            | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp1 | lprov | ETH/DEC20 | 100000            | 0.001 | buy  | BID              | 100        | 55     | amendmend  |
    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 5       | 10        | 10        | 3556         | 100000         | 10            |
      | 10         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 10      | 10        | 10        | 3556         | 100000         | 10            |

    When the parties place the following orders:
      | party      | market id  | side | volume | price | resulting trades | type       | tif     |
      | auxiliary2 | ETH/DEC20  | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2    | ETH/DEC20  | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader2 | ETH/DEC20 | 1026        | 1539   | 2052    | 3078    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 1026   | 0       |

    When the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 40    | 0                | TYPE_LIMIT | TIF_GTC |

   Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 29877        | 100000         | 11            |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader2 | -1     | 0              | 0            |

    Then the network moves ahead "14" blocks
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger               | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 29877        | 100000         | 11            |

    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 40         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 29877        | 100000         | 21            |   

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader2 | 0      | 0              | -1026        |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 0      | 0       |
   