Feature: Close potential positions

  Scenario: Cancel all order from party with only potential position release all margins
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
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

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
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |
    #And the cumulated balance for all accounts should be worth "4050075000"
    # setup order book
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux1             | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
      | party2           | ETH/DEC19 | buy  | 100    | 80    | 0                | TYPE_LIMIT | TIF_GTC | party2-b-1      |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 70    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # party1 maintenance margin: position*(mark_price*risk_factor_short+slippage_per_unit) + OrderVolume x Order_price x risk_factor_short  = 100 x 100 x 0.4878731  is about 4879

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | party1-s-1 |

    # party1 margin account: MarginInitialFactor x MaintenanceMarginLevel = 4879*1.5
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
      | party1 | ETH/DEC19 | sell | 300    | 120   | 0                | TYPE_LIMIT | TIF_GTC | party1-s-2 |

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
      | buy  | 70    | 1000   |

    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | aux2  | ETH/DEC19 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-5     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 21467       | 25760  | 32200   | 42934   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 29272  | 728     |

    ### At this point party1 have not traded and should have a position of 0
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 0            |

    And the mark price should be "110" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    ### Now we cancel the party1 orders
    Then the parties cancel the following orders:
      | party  | reference  |
      | party1 | party1-s-1 |

    ### balances are reduced (not)
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 29272  | 728     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 21467       | 25760  | 32200   | 42934   |

    ### cancel the last order
    Then the parties cancel the following orders:
      | party  | reference  |
      | party1 | party1-s-2 |

    And the network moves ahead "1" blocks

    ### balance are 0 out
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 0      | 30000   |

    ### still same margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 21467       | 25760  | 32200   | 42934   |

    ### then we place new orders and get a trade
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | sell | 1      | 130   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | aux2  | ETH/DEC19 | buy  | 1      | 130   | 1                | TYPE_LIMIT | TIF_GTC | ref-5     |

    ### balance are still 0
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 0      | 30000   |
