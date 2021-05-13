Feature: Test interactions between different auction types

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0     |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.004     | 0.001              | 0.3           |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2021-12-31T23:59:59Z |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |
    And the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | trader0 | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

  Scenario: When trying to exit opening auction liquidity monitoring doesn't get triggered, hence the opening auction uncrosses and market goes into continuous trading mode.

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy        | BID             | 500              | -10          |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell       | ASK             | 500              | 10           |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

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
  Given the traders submit the following liquidity provision:
    | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
    | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy        | BID             | 1                | -2           |
    | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy        | MID             | 2                | -1           |
    | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell       | ASK             | 1                | 2            |
    | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell       | MID             | 2                | 1            |

  And the traders place the following orders:
    | trader  | market id | side | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

  # Again, pointless to check this in auction
  # And the price monitoring bounds are []

  When the opening auction period ends for market "ETH/DEC21"
  # Perhaps the reason for extending could be changed to reflect which check actually failed
  # In this case, though, it's the orderbook status, which applies to all auctions alike
  # So the trigger being AUCTION_TRIGGER_OPENING is as accurate as any
  Then the market data for the market "ETH/DEC21" should be:
    | trading mode                 | auction trigger            |
    | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING    |

  And the traders place the following orders:
    | trader  | market id | side | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

  When the network moves ahead "1" blocks
  Then the auction ends with a traded volume of "10" at a price of "1000"

  And the market data for the market "ETH/DEC21" should be:
    | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
    | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 10000          | 10            |
  # how can we trade, and still be in auction?
  # And the market data for the market "ETH/DEC21" should be:
  #   | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest |
  #   | 1000       | TRADING_MODE_MONITORING_AUCTION | 1       | 990       | 1010      | 1000         | 10000          | 10            |


Scenario: When trying to exit opening auction liquidity monitoring is triggered due to insufficient supplied stake, hence the opening auction gets extended, the markets trading mode is TRADING_MODE_MONITORING_AUCTION and the trigger is AUCTION_TRIGGER_LIQUIDITY.

  Given the following network parameters are set:
    | name                                          | value |
    | market.liquidity.targetstake.triggering.ratio | 0.8   |

  And the traders submit the following liquidity provision:
    | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
    | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy        | BID             | 1                | -2           |
    | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy        | MID             | 2                | -1           |
    | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell       | ASK             | 1                | 2            |
    | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell       | MID             | 2                | 1            |

  And the traders place the following orders:
    | trader  | market id | side | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
    | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

  When the opening auction period ends for market "ETH/DEC21"
  # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
  Then the market data for the market "ETH/DEC21" should be:
    | trading mode                    | auction trigger           |
    | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    #| TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING    |

  And the traders submit the following liquidity provision:
    | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
    | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | buy        | BID             | 1                | -2           |
    | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | buy        | MID             | 2                | -1           |
    | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | sell       | ASK             | 1                | 2            |
    | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | sell       | MID             | 2                | 1            |

  When the network moves ahead "1" blocks
  Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"

  And the traders submit the following liquidity provision:
    | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
    | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | buy        | BID             | 1                | -2           |
    | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | buy        | MID             | 2                | -1           |
    | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | sell       | ASK             | 1                | 2            |
    | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | sell       | MID             | 2                | 1            |

  When the network moves ahead "1" blocks

  Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"

  And the traders submit the following liquidity provision:
    | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
    | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
    | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
    | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
    | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |

  When the network moves ahead "1" blocks

  Then the auction ends with a traded volume of "10" at a price of "1000"
  And the market data for the market "ETH/DEC21" should be:
    | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
    | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

Scenario: Once market is in continuous trading mode: post a GFN order that should trigger liquidty auction, appropriate event is sent and market in TRADING_MODE_MONITORING_AUCTION
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    When the network moves ahead "1" blocks
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader2 | ETH/DEC21 | sell | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 20     | 1010  | 0                | TYPE_LIMIT | TIF_FOK |
    And the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC21" should be:
     | trading mode                    | auction trigger             | target stake | supplied stake | open interest |
     | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 1000         | 1000           | 10            |
     #| TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 1000           | 10            |


Scenario: Once market is in continuous trading mode: post a GFN order that should trigger price auction, check that the order gets stopped, appropriate event is sent and market remains in TRADING_MODE_CONTINUOUS
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | no-reject |
      | trader1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GFN | reject-me |
    Then the following orders should be stopped:
      | trader  | market id | reason                                               |
      | trader1 | ETH/DEC21 | ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS |
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |


Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> leave auction mode
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    # If the order traded there'd be insufficient liquidity for the market to operate, hence the order doesn't trade
    # and the market enters a liquidity monitoring auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | trader2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    When the network moves ahead "1" blocks
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the traders cancel the following orders:
      | trader  | reference   |
      | trader1 | cancel-me-1 |
      | trader2 | cancel-me-2 |

    And debug market data for "ETH/DEC21"

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | sell       | MID             | 2                | 1            |

    # leave liquidity auction
    When the network moves ahead "2" blocks
    # should trigger price monitoring extension
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    # price monitoring extension ends
    # End price auction extension
    When the network moves ahead "301" blocks
    Then the auction ends with a traded volume of "20" at a price of "1020"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1       | 1010      | 1030      | 3060         | 3060           | 30            |

Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> extend with liquidity monitoring -> leave auction mode
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | trader2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the traders cancel the following orders:
      | trader  | reference   |
      | trader1 | cancel-me-1 |
      | trader2 | cancel-me-2 |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 3060              | 0.001 | sell       | MID             | 2                | 1            |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell       | MID             | 2                | 1            |

    Then the auction ends with a traded volume of "30" at a price of "1020"

    # end auction
    When the network moves ahead "301" blocks
    Then the auction ends with a traded volume of "20" at a price of "1020"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | 1       | 1010      | 1030      | 4080         | 5000           | 40            |

Scenario: Once market is in continuous trading mode: enter price monitoring auction -> extend with liquidity monitoring auction -> leave auction mode
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 1000              | 0.001 | sell       | MID             | 2                | 1            |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |
    And the network moves ahead "1" blocks

    # end price auction
    When the network moves ahead "300" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell       | MID             | 2                | 1            |

    When the network moves ahead "1" blocks
    Then the auction ends with a traded volume of "20" at a price of "1020"
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1020       | TRADING_MODE_CONTINUOUS | 1       | 1010      | 1030      | 3060         | 5000           | 20            |
