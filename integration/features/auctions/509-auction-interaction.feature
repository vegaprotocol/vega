Feature: Test interactions between different auction types

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0     |
      | network.floatingPointUpdates.delay            | 5s    |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2021-12-31T23:59:59Z |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

  Scenario: When trying to exit opening auction liquidity monitoring doesn't get triggered, hence the opening auction uncrosses and market goes into continuous trading mode.

    Given the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 500        | -10    |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 500        | 10     |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    # this is a bit pointless, we're still in auction, price bounds aren't checked
    # And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 10000          | 10            |

  Scenario: When trying to exit opening auction liquidity monitoring is triggered due to missing best bid, hence the opening auction gets extended, the markets trading mode is TRADING_MODE_MONITORING_AUCTION and the trigger is AUCTION_TRIGGER_LIQUIDITY.

  # This ought to be "buy_shape" and "sell_shape" equivalents
    Given the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

  # Again, pointless to check this in auction
  # And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC21"
  # Perhaps the reason for extending could be changed to reflect which check actually failed
  # In this case, though, it's the orderbook status, which applies to all auctions alike
  # So the trigger being AUCTION_TRIGGER_OPENING is as accurate as any
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks
    Then the auction ends with a traded volume of "10" at a price of "1000"

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 10000          | 10            |


  Scenario: When trying to exit opening auction liquidity monitoring is triggered due to insufficient supplied stake, hence the opening auction gets extended, the markets trading mode is TRADING_MODE_MONITORING_AUCTION and the trigger is AUCTION_TRIGGER_LIQUIDITY.

    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
  # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    #| TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING    |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks

    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

  Scenario: Once market is in continuous trading mode: post a GFN order that should trigger liquidty auction, appropriate event is sent and market in TRADING_MODE_MONITORING_AUCTION
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           | target stake | supplied stake | open interest |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 1000           | 10            |


  Scenario: Once market is in continuous trading mode: post a GFN order that should trigger price auction, check that the order gets stopped, appropriate event is sent and market remains in TRADING_MODE_CONTINUOUS
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                       |
      | party2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | no-reject |                                                             |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GFN | reject-me | OrderError: non-persistent order trades out of price bounds |
    Then the following orders should be stopped:
      | party  | market id | reason                                               |
      | party1 | ETH/DEC21 | ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS |
    
    Then the network moves ahead "5" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1001      | 1019      | 1000         | 1000           | 10            |


  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> leave auction mode
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    # If the order traded there'd be insufficient liquidity for the market to operate, hence the order doesn't trade
    # and the market enters a liquidity monitoring auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    When the network moves ahead "1" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties cancel the following orders:
      | party  | reference   |
      | party1 | cancel-me-1 |
      | party2 | cancel-me-2 |

    When the network moves ahead "1" blocks
    Then  the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | MID              | 2          | 1      |

    # leave liquidity auction
    When the network moves ahead "2" blocks
    # We should be able to leave liquidity auction now
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1030  | 0                | TYPE_LIMIT | TIF_GTC |

    # price monitoring extension ends
    # End price auction extension
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1       | 1010      | 1030      | 3060         | 4080           | 30            |

  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> extend with liquidity monitoring -> leave auction mode
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    And the network moves ahead "1" blocks

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties cancel the following orders:
      | party  | reference   |
      | party1 | cancel-me-1 |
      | party2 | cancel-me-2 |

    When the network moves ahead "2" blocks

    Then the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | MID              | 2          | 1      |

    # We're still in liquidity auction
    And the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    # We were in liquidity auction, we've updated the commitment amount
    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | party0 | ETH/DEC21 | 5100              | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | party0 | ETH/DEC21 | 5100              | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | party0 | ETH/DEC21 | 5100              | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | party0 | ETH/DEC21 | 5100              | 0.001 | sell | MID              | 2          | 1      |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "10" blocks

    # Now we place some orders that are outside the price range, auction is extended by price (300)
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    # jump ahead to the end of the auction
    When the network moves ahead "291" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | 1       | 1010      | 1030      | 4080         | 5100           | 40            |


      #Scenario: Once market is in continuous trading mode: enter price monitoring auction -> extend with liquidity monitoring auction -> leave auction mode
      #    Given the following network parameters are set:
      #      | name                                          | value |
      #      | market.liquidity.targetstake.triggering.ratio | 0.8   |
      #
      #    And the parties submit the following liquidity provision:
      #      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      #      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
      #      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
      #      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
      #      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |
      #
      #    And the parties place the following orders:
      #      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      #      | party1 | ETH/DEC21 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      #      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      #      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      #      | party2 | ETH/DEC21 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      #      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      #      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      #
      #    When the opening auction period ends for market "ETH/DEC21"
      #    Then the auction ends with a traded volume of "10" at a price of "1000"
      #    And the market data for the market "ETH/DEC21" should be:
      #     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      #     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |
      #
      #    When the parties place the following orders:
      #      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      #      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      #      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      #
      #    Then the market data for the market "ETH/DEC21" should be:
      #      | trading mode                    | auction trigger       |
      #      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |
      #    And the network moves ahead "1" blocks
      #
      #    # end price auction
      #    When the network moves ahead "300" blocks
      #    Then the market data for the market "ETH/DEC21" should be:
      #      | trading mode                    | auction trigger           |
      #      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
      #
      #    And the parties submit the following liquidity provision:
      #      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      #      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy        | BID             | 1                | -2           |
      #      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy        | MID             | 2                | -1           |
      #      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell       | ASK             | 1                | 2            |
      #      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell       | MID             | 2                | 1            |
      #
      #    When the network moves ahead "1" blocks
      #    Then the auction ends with a traded volume of "20" at a price of "1020"
      #    And the market data for the market "ETH/DEC21" should be:
      #     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      #     | 1020       | TRADING_MODE_CONTINUOUS | 1       | 1010      | 1030      | 3060         | 5000           | 20            |
