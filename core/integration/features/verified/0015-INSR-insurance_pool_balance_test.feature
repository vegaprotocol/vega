Feature: Test closeout type 1: margin >= cost of closeout

  Scenario: case 1 (using simple risk model) from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910 (0015-INSR-001, 0015-INSR-003, 0018-RSKM-001, 0018-RSKM-003, 0010-MARG-004, 0010-MARG-005, 0010-MARG-006, 0010-MARG-007, 0010-MARG-008. 0010-MARG-009)
  Background:

    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 1    | 2     | 100         | -100          | 0.1                    |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.3           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | USD        | USD   | simple-risk-model-1 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    # setup accounts

    Given the insurance pool balance should be "0" for the market "ETH/DEC19"
    Given the initial insurance pool balance is "15000" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount     |
      | sellSideProvider | USD   | 1000000000 |
      | buySideProvider  | USD   | 1000000000 |
      | party1           | USD   | 30000      |
      | party2           | USD   | 50000000   |
      | party3           | USD   | 30500      |
      | aux1             | USD   | 1000000000 |
      | aux2             | USD   | 1000000000 |
      | lpprov           | USD   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # setup order book
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux1             | ETH/DEC19 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux1             | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux2             | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general   |
      | aux1             | USD   | ETH/DEC19 | 1200   | 999998800 |
      | aux2             | USD   | ETH/DEC19 | 180    | 999999820 |
      | sellSideProvider | USD   | ETH/DEC19 | 450000 | 999550000 |
      | buySideProvider  | USD   | ETH/DEC19 | 120000 | 999880000 |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general   |
      | aux1             | USD   | ETH/DEC19 | 675    | 999999325 |
      | aux2             | USD   | ETH/DEC19 | 330    | 999999670 |
      | sellSideProvider | USD   | ETH/DEC19 | 300000 | 999700000 |
      | buySideProvider  | USD   | ETH/DEC19 | 150000 | 999850000 |
    # margin_sellSideProvider: 1000*100*2*1.5=300000
    # margin_buySideProvider: 1000*100*1*1.5=150000

    # party 1 place an order + we check margins
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 30000  | 0       |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/DEC19 | 20000       | 30000   |
    # margin_party1 = 100*100*2=20000

    # all general acc balance goes to margin account for the order, 'party1' should have 100*100*3
    # in the margin account as its Position*Markprice*Initialfactor

    # then party2 places an order, this trades with party1 and we calculate the margins again
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 100    | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the mark price should be "100" for the market "ETH/DEC19"
    And the insurance pool balance should be "15000" for the market "ETH/DEC19"

    #check margin account and margin level
    # And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | USD   | ETH/DEC19 | 30000  | 0        |
      | party2 | USD   | ETH/DEC19 | 30000  | 49969000 |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/DEC19 | 25000       | 37500   |
      | party2 | ETH/DEC19 | 12000       | 18000   |
    #margin_party1: 100*(150-100)+100*2*100=25000

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 300   | 1      |
      | sell | 150   | 1000   |
      | sell | 160   | 563    |
      | buy  | 80    | 1000   |
      | buy  | 70    | 1286   |
      | buy  | 20    | 1      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 1      | 126   | 0                | TYPE_LIMIT | TIF_GTC | ref-1-xxx |

    # Margin account balance brought up to new initial level as order is placed (despite all balance being above search level)
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | USD   | ETH/DEC19 | 30000  | 0        |
      | party2 | USD   | ETH/DEC19 | 18150  | 49980850 |
    # New margin level calculated after placing an order
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/DEC19 | 25000       | 37500   |
      | party2 | ETH/DEC19 | 12100       | 18150   |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 300   | 1      |
      | sell | 160   | 563    |
      | sell | 150   | 1000   |
      | buy  | 126   | 1      |
      | buy  | 116   | 776    |
      | buy  | 80    | 1000   |
      | buy  | 20    | 1      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | sell | 1      | 126   | 1                | TYPE_LIMIT | TIF_GTC | ref-1-xxx |
    Then the mark price should be "126" for the market "ETH/DEC19"
    And the insurance pool balance should be "38500" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 300   | 1      |
      | sell | 160   | 563    |
      | sell | 150   | 900    |
      | buy  | 126   | 0      |
      | buy  | 116   | 0      |
      | buy  | 80    | 1000   |
      | buy  | 70    | 1286   |
      | buy  | 20    | 1      |

    # Margin account balance not updated following a trade (above search)
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party2 | ETH/DEC19 | 17372       | 26058   |

    # margin_party2: 101*(126-80)+101*1*126=17372

    # MTM win transfer
    Then the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount | asset |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 2600   | USD   |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 150   | 900    |
      | sell | 300   | 1      |
      | buy  | 80    | 1000   |
      | buy  | 20    | 1      |

    Then the mark price should be "126" for the market "ETH/DEC19"
    And the insurance pool balance should be "38500" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general   |
      | party1           | USD   | ETH/DEC19 | 0      | 0         |
      | party2           | USD   | ETH/DEC19 | 26058  | 49975542  |
      | party3           | USD   | ETH/DEC19 | 414    | 30073     |
      | aux1             | USD   | ETH/DEC19 | 792    | 999999182 |
      | aux2             | USD   | ETH/DEC19 | 447    | 999999579 |
      | sellSideProvider | USD   | ETH/DEC19 | 380400 | 999622000 |
      | buySideProvider  | USD   | ETH/DEC19 | 189000 | 999811000 |

    # margin_sellSideProvider: 1000*126*2*3=756000
    # margin_buySideProvider: 1000*126*1*3=378000
    Then the parties should have the following margin levels:
      #check margin account and margin level
      | party  | market id | maintenance | initial |
      | party1 | ETH/DEC19 | 0           | 0       |
      | party2 | ETH/DEC19 | 17372       | 26058   |
      | party3 | ETH/DEC19 | 276         | 414     |

    And the cumulated balance for all accounts should be worth "5050075500"
    And the insurance pool balance should be "38500" for the market "ETH/DEC19"

    # order book volume change
    Then the parties cancel the following orders:
      | party            | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |

    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 500   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 20    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    #check margin account and margin level
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | USD   | ETH/DEC19 | 26058  | 49975542 |
      | party3 | USD   | ETH/DEC19 | 414    | 30073    |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial | 
      | party3 | ETH/DEC19 | 276         | 414     | 
      | party2 | ETH/DEC19 | 17372       | 26058   | 

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 50     | 30    | 0                | TYPE_LIMIT | TIF_GTC |

    #check margin account and margin level
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | USD   | ETH/DEC19 | 44598  | 49957002 |
      | party3 | USD   | ETH/DEC19 | 414    | 30073    |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party2 | ETH/DEC19 | 29732       | 44598   |
      | party3 | ETH/DEC19 | 276         | 414     |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 50     | 30    | 1                | TYPE_LIMIT | TIF_GTC |
    And the insurance pool balance should be "38500" for the market "ETH/DEC19"
    Then the mark price should be "30" for the market "ETH/DEC19"
      # Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 500   | 1000   |
      | sell | 300   | 1      |
      | buy  | 20    | 1001   |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party3 | -51    | 96             | 0            |

    #check margin account and margin level
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | USD   | ETH/DEC19 | 9060   | 49982844 |
      | party3 | USD   | ETH/DEC19 | 25933  | 4500     |

    # party3 maintenance margin: position*(mark_price*risk_factor_short+slippage_per_unit) + mark_price*risk_factor_short=51*(30*2+466)+0=26826
    # (slippage calulated as follows) slippager_per_unit=exit_price-mark_price=(300*1+500*50)/51-30=496-30=466
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party2 | ETH/DEC19 | 6040        | 9060    |
      | party3 | ETH/DEC19 | 17289       | 25933   |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 500   | 1000   |
      | sell | 300   | 1      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 50     | 30    | 0                | TYPE_LIMIT | TIF_GTC |

    # party3 maintenance margin: position*(mark_price*risk_factor_short+slippage_per_unit) + open_order*mark_price*risk_factor_short=51*(30*2+466) + 50 * 30 * 2 = 26826 + 3000 = 29826
    # (slippage calulated as follows) slippager_per_unit=exit_price-mark_price=(300*1+500*50)/51-30=496-30=466

    # party3 has put the order twice
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party3 | ETH/DEC19 | 20289       | 30433   |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 500   | 1000   |
      | sell | 300   | 1      |
      | sell | 30    | 50     |
      | buy  | 20    | 1001   |

    And the insurance pool balance should be "38500" for the market "ETH/DEC19"

    #check margin account and margin level
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party2 | USD   | ETH/DEC19 | 9060   | 49982844 |
      | party3 | USD   | ETH/DEC19 | 30433  | 0        |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party2 | ETH/DEC19 | 6040        | 9060    |
      | party3 | ETH/DEC19 | 20289       | 30433   |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 50     | 30    | 1                | TYPE_LIMIT | TIF_GTC | ref-2-xxx |

    And the insurance pool balance should be "37533" for the market "ETH/DEC19"

    #party3 gets closeout with MTM
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | USD   | ETH/DEC19 | 0      | 0        |
      | party2 | USD   | ETH/DEC19 | 11310  | 49980444 |
      | party3 | USD   | ETH/DEC19 | 0      | 0        |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | -30000       |
      | party2 | 201    | -7096          | 0            |
      | party3 | 0      | 0              | -30337       |

  Scenario: case 2 using lognomal risk model (0015-INSR-003, 0010-MARG-009, 0010-MARG-010, 0010-MARG-011)
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
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 0.001                  | 0                         |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount     |
      | sellSideProvider | USD   | 1000000000 |
      | buySideProvider  | USD   | 1000000000 |
      | party1           | USD   | 30000      |
      | party2           | USD   | 50000000   |
      | party3           | USD   | 30000      |
      | aux1             | USD   | 1000000000 |
      | aux2             | USD   | 1000000000 |
      | lpprov           | USD   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |
    #And the cumulated balance for all accounts should be worth "4050075000"
    # setup order book
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux1             | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
      | party2           | ETH/DEC19 | buy  | 100    | 80    | 0                | TYPE_LIMIT | TIF_GTC | party2-b-1      |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 70    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | party1-s-1 |

    # party1 margin account: MarginInitialFactor x MaintenanceMarginLevel = 4879*1.5=7318
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 7318   | 22682   |

    # party1 maintenance margin level: position*(mark_price*risk_factor_short+slippage_per_unit) + OrderVolume x Mark_price x risk_factor_short  = 100 x 100 x 0.4878731  is about 4879
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 4879        | 5854   | 7318    | 9758    |

    # party1 place more order volume 300
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC19 | sell | 300    | 120   | 0                | TYPE_LIMIT | TIF_GTC | party1-s-1 |

    # party1 maintenance margin level: position*(mark_price*risk_factor_short+slippage_per_unit) + OrderVolume x Mark_price x risk_factor_short  = 100 x 400 x 0.4878731  is about 19515
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 29272  | 728     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 19515       | 23418  | 29272   | 39030   |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 150   | 1000   |
      | sell | 120   | 400    |
      | buy  | 80    | 100    |
      | buy  | 70    | 2286   |

    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    #########################################
    #MTM closeout party1
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | aux2  | ETH/DEC19 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-5     |

    # margin on order should be mark_price x volume x rf = 110 x 400 x 0.4878731 = 21466
    # margin account is above maintenance level, so it stays at 29272
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 29272  | 728     |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 21467       | 25760  | 32200   | 42934   |

    And the mark price should be "110" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 119   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | aux2  | ETH/DEC19 | sell | 1      | 119   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "119" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 150   | 1000   |
      | sell | 120   | 400    |
      | buy  | 80    | 100    |
      | buy  | 70    | 2286   |

    # margin on order should be mark_price x volume x rf = 119 x 400 x 0.4878731 = 23223
    # margin account is above maintenance level, so it stays at 29272
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 29272  | 728     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 23223       | 27867  | 34834   | 46446   |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 0            |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | aux2  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 29272  | 728     |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 23418       | 28101  | 35127   | 46836   |


