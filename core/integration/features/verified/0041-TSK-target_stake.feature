Feature: Target stake
  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    # Market risk parameters and assets don't really matter.
    # We need to track open interest i.e. sum of all long positions across the parties and how they change over time
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r  | sigma |
      | 0.000001      | 0.00011407711613050422 | -1 | -1 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.00025   | 0.0005             |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.1           | 1.2            | 1.4            |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the average block duration is "1"

    # T0
    And time is updated to "2021-03-08T00:00:00Z"

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount    |
      | lp_1  | BTC   | 100000000 |
      | lp_2  | BTC   | 100000000 |
      | lp_3  | BTC   | 100000000 |
      | lp_4  | BTC   | 100000000 |
      | lp_5  | BTC   | 100000000 |
      | lp_6  | BTC   | 100000000 |
      | lp_7  | BTC   | 100000000 |
      | lp_8  | BTC   | 100000000 |
      | tt_1  | BTC   | 100000000 |
      | tt_2  | BTC   | 100000000 |
      | tt_3  | BTC   | 100000000 |
      | tt_4  | BTC   | 100000000 |

  Scenario: Max open interest changes over time (0041-TSTK-002, 0041-TSTK-003, 0042-LIQF-007)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.0              | 10s         | 1.5            |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 1e6                    | 1e6                       |

    # put some volume on the book so that others can increase their
    # positions and close out if needed too
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp_1  | ETH/DEC21 | buy  | 1000   | 90    | 0                | TYPE_LIMIT | TIF_GTC | lp_1_0    |
      | lp_1  | ETH/DEC21 | sell | 1000   | 110   | 0                | TYPE_LIMIT | TIF_GTC | lp_1_1    |

    # nothing should have traded yet
    Then the mark price should be "0" for the market "ETH/DEC21"

    # Traders 1, 2, 3 go long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_1_0    |
      | tt_2  | ETH/DEC21 | buy  | 20     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |
      | tt_3  | ETH/DEC21 | buy  | 30     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |

    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp_1  | ETH/DEC21 | 135               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | lp_1  | ETH/DEC21 | 135               | 0.001 | sell | ASK              | 1          | 10     | amendment  |
      | lp2 | lp_2  | ETH/DEC21 | 165               | 0.002 | buy  | BID              | 1          | 10     | submission |
      | lp2 | lp_2  | ETH/DEC21 | 165               | 0.002 | sell | ASK              | 1          | 10     | amendment  |
      | lp3 | lp_3  | ETH/DEC21 | 300               | 0.003 | buy  | BID              | 1          | 10     | submission |
      | lp3 | lp_3  | ETH/DEC21 | 300               | 0.003 | sell | ASK              | 1          | 10     | amendment  |
      | lp4 | lp_4  | ETH/DEC21 | 300               | 0.004 | buy  | BID              | 1          | 10     | submission |
      | lp4 | lp_4  | ETH/DEC21 | 300               | 0.004 | sell | ASK              | 1          | 10     | amendment  |
      | lp5 | lp_5  | ETH/DEC21 | 500               | 0.005 | buy  | BID              | 1          | 10     | submission |
      | lp5 | lp_5  | ETH/DEC21 | 500               | 0.005 | sell | ASK              | 1          | 10     | amendment  |
      | lp6 | lp_6  | ETH/DEC21 | 300               | 0.006 | buy  | BID              | 1          | 10     | submission |
      | lp6 | lp_6  | ETH/DEC21 | 300               | 0.006 | sell | ASK              | 1          | 10     | amendment  |
      | lp7 | lp_7  | ETH/DEC21 | 200               | 0.007 | buy  | BID              | 1          | 10     | submission |
      | lp7 | lp_7  | ETH/DEC21 | 200               | 0.007 | sell | ASK              | 1          | 10     | amendment  |
      | lp8 | lp_8  | ETH/DEC21 | 100               | 0.008 | buy  | BID              | 1          | 10     | submission |
      | lp8 | lp_8  | ETH/DEC21 | 100               | 0.008 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"

    # Now parties 1,2,3 are long so open intereset = 10+20+30 = 60.
    # Target stake is mark_price x max_oi x target_stake_scaling_factor x rf_short
    # rf_short should have been set above to 0.1
    # target_stake = 110 x 60 x 1.5 x 0.1
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 990          | 2000           | 60            |
    And the liquidity fee factor should be "0.005" for the market "ETH/DEC21"
    # T0 + 1s
    Then the network moves ahead "1" blocks

    # Trader 3 closes out 20
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_3  | ETH/DEC21 | sell | 20     | 90    | 1                | TYPE_LIMIT | TIF_GTC | tt_2_1    |

    # the maximum oi over the last 10s is still unchanged
    # target_stake = 90 x 60 x 1.5 x 0.1 = 810
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 90         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 810          | 2000           | 40            |
    And the liquidity fee factor should be "0.004" for the market "ETH/DEC21"

    # T0 + 10s
    Then the network moves ahead "10" blocks
    # now the peak of 60 should have passed from window
    # target_stake = 90 x 40 x 1.5 x 0.1 = 540
    And the target stake should be "540" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_3  | ETH/DEC21 | sell | 10     | 90    | 1                | TYPE_LIMIT | TIF_GTC | tt_2_1    |

    # target stake should be: 90 x 40 x 1.5 x 0.1 = 540
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 90         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 540          | 2000           | 30            |
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"

    # target stake should be: 90 x 40 x 1.5 x 0.1 = 540
    And the target stake should be "540" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"

    Then the network moves ahead "1" blocks
    # target stake should be: 90 x 30 x 1.5 x 0.1 = 405
    And the target stake should be "405" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"

    # Move time 4x the window, the target stake remain unchanged as we rely on last value even if it drops outside the window (that's still what OI is)
    Then the network moves ahead "40" blocks
    And the target stake should be "405" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | tt_1  | ETH/DEC21 | sell | 10     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    # target stake is: 90 x 20 x 1.5 x 0.1 = 270 as now the max OI within the window is 20
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 90         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 270          | 2000           | 20            |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | tt_2  | ETH/DEC21 | sell | 20     | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    # OI is now 0, but target stake remains unchanged as max OI of 20 is still within the window
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 90         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 270          | 2000           | 0             |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    # target stake remain unchanged as open interest of 20 and 0 where both recorded with the same timestamp
    Then the network moves ahead "10" blocks
    And the target stake should be "270" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     |
      | tt_2  | ETH/DEC21 | sell | 5      | 0     | 1                | TYPE_MARKET | TIF_FOK |
      | tt_2  | ETH/DEC21 | sell | 5      | 0     | 1                | TYPE_MARKET | TIF_FOK |

    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 90         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 135          | 2000           | 10            |
    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     |
      | tt_2  | ETH/DEC21 | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_FOK |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | 110               | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 165          | 2000           | 0             |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    # O is now the last recorded open interest so target stake should drop to 0
    Then the network moves ahead "10" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 0            | 2000           | 0             |
    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

  Scenario: Max open interest changes over time, testing change of timewindow (0041-TSTK-001; 0041-TSTK-004; 0041-TSTK-005)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.0              | 20s         | 1.5            |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 1e6                    | 1e6                       |

    # put some volume on the book so that others can increase their
    # positions and close out if needed too
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp_1  | ETH/DEC21 | buy  | 1000   | 90    | 0                | TYPE_LIMIT | TIF_GTC | lp_1_0    |
      | lp_1  | ETH/DEC21 | sell | 1000   | 110   | 0                | TYPE_LIMIT | TIF_GTC | lp_1_1    |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp_1  | ETH/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 20     | submission |
      | lp1 | lp_1  | ETH/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | -20    | submission |

    # nothing should have traded, we have mark price set apriori or
    # due to auction closing.
    Then the mark price should be "0" for the market "ETH/DEC21"

    # Traders 1, 2, 3 go long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_1_0    |
      | tt_2  | ETH/DEC21 | buy  | 20     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |
      | tt_3  | ETH/DEC21 | buy  | 30     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |

    Then the opening auction period ends for market "ETH/DEC21"

    # So now parties 1,2,3 are long 10+20+30 = 60 open interest.
    # Target stake is mark_price x max_oi x target_stake_scaling_factor x rf_short
    # rf_short should have been set above to 0.1
    # target_stake = 110 x 60 x 1.5 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | 990          | 2000           | 60            |

    # T0 + 1s
    Then the network moves ahead "1" blocks

    # Trader 3 closes out 20
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_3  | ETH/DEC21 | sell | 20     | 90    | 1                | TYPE_LIMIT | TIF_GTC | tt_2_1    |

    # the maximum oi over the last 20s is still unchanged
    # target_stake = 90 x 60 x 1.5 x 0.1 = 810
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 90         | 90                | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 810          | 2000           | 40            |

    # T0 + 10s
    Then the network moves ahead "10" blocks
    # the max_io stays the same as previous timestep as the timeWindow is bigger in this scenario (20s days instead of 10s)
    # target_stake = 90 x 60 x 1.5 x 0.1 = 810
    And the target stake should be "810" for the market "ETH/DEC21"

    Then the network moves ahead "10" blocks
    # target_stake = 90 x 40 x 1.5 x 0.1 = 540
    And the target stake should be "540" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | buy  | 100    | 110   | 1                | TYPE_LIMIT | TIF_GTC | lp_1_0    |
    Then the mark price should be "110" for the market "ETH/DEC21"

    # max_io=10+20+30-20+100=140
    # target_stake = 110 x 140 x 1.5 x 0.1=2310
    And the target stake should be "2310" for the market "ETH/DEC21"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.0              | 10s         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 1e6                    | 1e6                       |

    # target_stake = 110 x 140 x 1 x 0.1 =1540
    And the target stake should be "1540" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | buy  | 30     | 110   | 1                | TYPE_LIMIT | TIF_GTC | tt_1_0    |

    # target_stake = 110 x (140+30) x 170 x 1 x 0.1=1870
    And the target stake should be "1870" for the market "ETH/DEC21"

  Scenario: Target stake is calculate correctly during auction in presence of wash trades

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.0              | 10s         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 1e6                    | 1e6                       |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp_1  | ETH/DEC21 | buy  | 1000   | 90    | 0                | TYPE_LIMIT | TIF_GTC | lp_1_0    |
      | lp_1  | ETH/DEC21 | sell | 1000   | 200   | 0                | TYPE_LIMIT | TIF_GTC | lp_1_1    |

    Then the mark price should be "0" for the market "ETH/DEC21"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | sell | 50     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_1_0    |
      | tt_2  | ETH/DEC21 | buy  | 20     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |
      | tt_3  | ETH/DEC21 | buy  | 30     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |

    And the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 550          | 0              | 0             |

    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp_1  | ETH/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | lp_1  | ETH/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    # Add wash trades
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_4  | ETH/DEC21 | sell | 40     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_4_0    |
      | tt_4  | ETH/DEC21 | buy  | 40     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_4_1    |

    # Check that target stake is unchanged
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 550          | 2000           | 0             |

    Then the opening auction period ends for market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 550          | 2000           | 50            |

  Scenario: Target stake can drop during auction

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 1.0              | 10s         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 1e6                    | 1e6                       |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp_1  | ETH/DEC21 | buy  | 1000   | 90    | 0                | TYPE_LIMIT | TIF_GTC | lp_1_0    |
      | lp_1  | ETH/DEC21 | sell | 1000   | 200   | 0                | TYPE_LIMIT | TIF_GTC | lp_1_1    |
    Then the mark price should be "0" for the market "ETH/DEC21"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | sell | 50     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_1_0    |
      | tt_2  | ETH/DEC21 | buy  | 20     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |
      | tt_3  | ETH/DEC21 | buy  | 30     | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |
    And the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 550          | 0              | 0             |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp_1  | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | lp_1  | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 10     | amendment  |
    And the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 550          | 1000           | 50            |

    When the network moves ahead "5" blocks
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | tt_1  | ETH/DEC21 | sell | 50     | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | tt_2  | ETH/DEC21 | buy  | 50     | 110   | 1                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1100         | 1000           | 100           |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 1100         | 1000           | 100           |

    When the network moves ahead "11" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 1100         | 1000           | 100           |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | tt_1  | ETH/DEC21 | buy  | 50     | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | tt_2  | ETH/DEC21 | sell | 50     | 110   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 550          | 1000           | 100           |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 110        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 550          | 1000           | 50            |
