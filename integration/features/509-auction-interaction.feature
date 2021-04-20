Feature: Test interactions between different auction types

  Background:
    Given the following network parameters are set:
      | name                              | value |
      | market.stake.target.timeWindow    | 86400 |
      | market.stake.target.scalingFactor | 1     |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.004     | 0.001              | 0.3           |
    And the price monitoring updated every "0" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2021-12-31T23:59:59Z |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |
    And the liquidity order collection object with reference "buy_shape":
      | reference | offet | proportion |
      | BEST_BID  | -2    | 1          |
      | MID       | -1    | 2          |
    And the liquidity order collection object with reference "sell_shape":
      | reference | offet | proportion |
      | BEST_ASK  | 2     | 1          |
      | MID       | 1     | 2          |
    And the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | trader0 | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

  Scenario: When trying to exit opening auction liquidity monitoring doesn't get triggered, hence the opening auction uncrosses and market goes into continuous trading mode.

    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0     |


    And the traders submit the following liquidity provision:
      | id  | trader  | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC19 | 10000             | 0.001 | buy        | BID             | 500              | -10          |
      | lp1 | trader0 | ETH/DEC19 | 10000             | 0.001 | sell       | ASK             | 500              | 10           |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 10000

  Scenario: When trying to exit opening auction liquidity monitoring is triggered due to missing best bid, hence the opening auction gets extended, the markets trading mode is TRADING_MODE_MONITORING_AUCTION and the trigger is AUCTION_TRIGGER_LIQUIDITY.
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0     |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 10000             | 0.001   | "buy_shape"      | "sell_shape"      |
    Then the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC19 | 10000             | 0.001 | buy        | BID             | 500              | -10        |
      | lp1 | trader0 | ETH/DEC19 | 10000             | 0.001 | sell       | ASK             | 500              | 10         |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction for market "ETH/DEC19" gets extended with the "AUCTION_TRIGGER_LIQUIDITY" trigger
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    And the auction ends resulting in traded volume of "10" at a price of "1000"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 10000

  Scenario: When trying to exit opening auction liquidity monitoring is triggered due to insufficient supplied stake, hence the opening auction gets extended, the markets trading mode is TRADING_MODE_MONITORING_AUCTION and the trigger is AUCTION_TRIGGER_LIQUIDITY.

    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 700               | 0.001   | "buy_shape"      | "sell_shape"      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction for market "ETH/DEC19" gets extended with the "AUCTION_TRIGGER_LIQUIDITY" trigger
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 800               | 0.001   | "buy_shape"      | "sell_shape"      |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 801               | 0.001   | "buy_shape"      | "sell_shape"      |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 1000              | 0.001   | "buy_shape"      | "sell_shape"      |

    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 10000

  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> leave auction mode
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 1000              | 0.001   | "buy_shape"      | "sell_shape"      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 1000

    # If the order traded there'd be insufficient liquidity for the market to operate, hence the order doesn't trade
    # and the market enters a liquidity monitoring auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader1 | ETH/DEC19 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | trader2 | ETH/DEC19 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    And the auction for market "ETH/DEC19" gets started with the "AUCTION_TRIGGER_LIQUIDITY" trigger
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_MONITORING_AUCTION"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the traders cancel the following orders:
      | trader  | reference   |
      | trader1 | cancel-me-1 |
      | trader2 | cancel-me-2 |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 3060              | 0.001   | "buy_shape"      | "sell_shape"      |

    And the time is advance beyond "min_auction_length"
    Then the auction for market "ETH/DEC19" gets started with the "AUCTION_TRIGGER_PRICE" trigger
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the auction duration is "3s"

    When the time is advanced by "4s"
    Then the auction ends resulting in traded volume of "20" at a price of "1020"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "30"
    And the mark price should be "1020" for the market "ETH/DEC21"
    And the price monitoring bounds are [[1010,1030]]
    And the target stake should be "3060" for the market "ETH/DEC21"
    And the supplied stake is 3060

  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> extend with liquidity monitoring -> leave auction mode

    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 1000              | 0.001   | "buy_shape"      | "sell_shape"      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 1000

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader1 | ETH/DEC19 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | trader2 | ETH/DEC19 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    Then the auction for market "ETH/DEC19" gets started with the "AUCTION_TRIGGER_LIQUIDITY" trigger
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the traders cancel the following orders:
      | trader  | reference   |
      | trader1 | cancel-me-1 |
      | trader2 | cancel-me-2 |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 3060              | 0.001   | "buy_shape"      | "sell_shape"      |

    Then the auction for market "ETH/DEC19" gets started with the "AUCTION_TRIGGER_PRICE" trigger
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the auction duration is "3s"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the time is advanced by "4s"
    Then the auction for market "ETH/DEC19" gets extended with the "AUCTION_TRIGGER_LIQUIDITY" trigger
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_MONITORING_AUCTION"

    When traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 5000              | 0.001   | "buy_shape"      | "sell_shape"      |

    Then the auction ends resulting in traded volume of "30" at a price of "1020"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "40"
    And the mark price should be "1020" for the market "ETH/DEC21"
    And the price monitoring bounds are [[1010,1030]]
    And the target stake should be "4080" for the market "ETH/DEC21"
    And the supplied stake is 5000

  Scenario: Once market is in continuous trading mode: enter price monitoring auction -> extend with liquidity monitoring auction -> leave auction mode

    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 1000              | 0.001   | "buy_shape"      | "sell_shape"      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 1000

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader1 | ETH/DEC19 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | trader2 | ETH/DEC19 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    Then the auction for market "ETH/DEC19" gets started with the "AUCTION_TRIGGER_PRICE" trigger
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the auction duration is "3s"

    When the time is advanced by "4s"
    Then the auction for market "ETH/DEC19" gets extended with the "AUCTION_TRIGGER_LIQUIDITY" trigger
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_MONITORING_AUCTION"

    When traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 5000              | 0.001   | "buy_shape"      | "sell_shape"      |

    Then the auction ends resulting in traded volume of "20" at a price of "1020"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the mark price should be "1020" for the market "ETH/DEC21"
    And the price monitoring bounds are [[1010,1030]]
    And the target stake should be "3060" for the market "ETH/DEC21"
    And the supplied stake is 5000

  Scenario: Once market is in continuous trading mode: post a GFN order that should trigger liquidity auction, check that the order gets rejected, appropriate event is sent and market remains in TRADING_MODE_CONTINUOUS
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And traders place following liquidity provisions:
      | trader  | market id | commitment amount | fee bid | buy shape object | sell shape object |
      | trader0 | ETH/DEC19 | 1000              | 0.001   | "buy_shape"      | "sell_shape"      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC19"
    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 1000

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GFN |           |
      | trader2 | ETH/DEC19 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | reject-me |
    And the order with reference "reject-me" gets rejected
    And the event informing that non-persistent order was rejected due to violating trigger "AUCTION_TRIGGER_LIQUIDITY"

    Then the auction ends resulting in traded volume of "10" at a price of "1000"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the max_oi for the market "ETH/DEC21" is "10"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the price monitoring bounds are [[990,1010]]
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake is 1000