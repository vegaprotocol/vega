Feature: Test liquidity monitoring

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.floatingPointUpdates.delay      | 10s   |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 4     |
    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 1s          | 1.0            |
    And the average block duration is "1"
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.0           | 1.0            | 2              |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99        | 300               |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH        | ETH   | lqm-params           | simple-risk-model-1     | margin-calculator-1       | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         | default-futures |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future | 0.5                    | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 100000000  |
      | lprov1 | ETH   | 1000000000 |
      | lprov2 | ETH   | 1000000000 |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | party4 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | lprov1 | USD   | 500000    |
      | lprov2 | USD   | 500000    |

  Scenario: 001: A market which enters a state requiring liquidity auction through increased open interest during a block but then leaves state again prior to block completion never enters liquidity auction. (0035-LIQM-005)
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | submission |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov1 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lprov1 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 2      |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC21 | buy  | 20     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 20     | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    Then clear all events

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 20     | 1010  | 2                | TYPE_LIMIT | TIF_GTC |

    # verify that we don't enter liquidity auction immediately despite liquidity being undersuplied
    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3000         | 1000           | 30            |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | lprov1 | ETH/DEC21 | 10000             | 0.001 | amendment |
      | lp1 | lprov1 | ETH/DEC21 | 10000             | 0.001 | amendment |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov1 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | lprov1 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 2      |
    # move to the next block to perform liquidity check, we should still be in continuous trading
    Then the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3030         | 10000          | 30            |

    # verify that at no point auction has been entered
    Then the following events should NOT be emitted:
      | type         |
      | AuctionEvent |

  Scenario: 002: A market which enters a state requiring liquidity auction through reduced current stake (e.g. through LP bankruptcy) during a block but then leaves state again prior to block completion never enters liquidity auction. (0035-LIQM-006)

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lprov1 | ETH/MAR22 | 50000             | 0.001 | submission |
      | lp1 | lprov1 | ETH/MAR22 | 50000             | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov1 | ETH/MAR22 | 2         | 1                    | sell | ASK              | 500    | 20     |
      | lprov1 | ETH/MAR22 | 2         | 1                    | buy  | BID              | 500    | 20     |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 35569        | 50000          | 10            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    And clear all events
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 142276       | 50000          | 40            |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp2 | lprov2 | ETH/MAR22 | 92276             | 0.001 | submission |
      | lp2 | lprov2 | ETH/MAR22 | 92276             | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov2 | ETH/MAR22 | 2         | 1                    | sell | ASK              | 100    | 20     |
      | lprov2 | ETH/MAR22 | 2         | 1                    | buy  | BID              | 100    | 20     |

    # move to the next block to perform liquidity check, we should still be in continuous trading
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 142276       | 142276         | 40            |

    # verify that at no point auction has been entered
    Then the following events should NOT be emitted:
      | type         |
      | AuctionEvent |

  # perhaps a bit pointless, liquidity auction was removed, keeping the scenario to make sure everything else works as expected.
  Scenario: 004: When the Max Open Interest field decreases for a created block to a level such that a liquidity auction which is active at the start of a block can now be exited the block stays in auction within the block but leaves at the end. (0035-LIQM-008)

    Given the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 1     |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 1.0              | 5s          | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties deposit on asset's general account the following amount:
      | party          | asset | amount |
      | lprov1         | ETH   | 1000   |
      | lp2Bdistressed | ETH   | 101    |
    And the parties submit the following liquidity provision:
      | id  | party          | market id | commitment amount | fee   | lp type    |
      | lp1 | lprov1         | ETH/DEC21 | 5999              | 0.001 | submission |
      | lp1 | lprov1         | ETH/DEC21 | 5999              | 0.001 |            |
      | lp2 | lp2Bdistressed | ETH/DEC21 | 1                 | 0.001 | submission |
      | lp2 | lp2Bdistressed | ETH/DEC21 | 1                 | 0.001 |            |
    And the parties place the following pegged iceberg orders:
      | party          | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lprov1         | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 2      | lp1_buy   |
      | lprov1         | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 2      | lp1_sell  |
      | lp2Bdistressed | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 10     | lp2_buy   |
      | lp2Bdistressed | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 10     | lp2_sell  |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC21 | buy  | 100    | 950   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 199    | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | open interest | best static bid price | static mid price | best static offer price | target stake | supplied stake |
      | 1000       | TRADING_MODE_CONTINUOUS | 60            | 970                   | 1000             | 1030                    | 6000         | 6000           |

    When the network moves ahead "1" blocks
    Then the orders should have the following states:
      | party          | market id | side | volume | remaining | price | status        | reference |
      | lprov1         | ETH/DEC21 | buy  | 1      | 1         | 968   | STATUS_ACTIVE | lp1_buy   |
      | lprov1         | ETH/DEC21 | sell | 1      | 1         | 1032  | STATUS_ACTIVE | lp1_sell  |
      | lp2Bdistressed | ETH/DEC21 | buy  | 1      | 1         | 960   | STATUS_ACTIVE | lp2_buy   |
      | lp2Bdistressed | ETH/DEC21 | sell | 1      | 1         | 1040  | STATUS_ACTIVE | lp2_sell  |
    And the parties should have the following margin levels:
      | party          | market id | maintenance | initial |
      | lp2Bdistressed | ETH/DEC21 | 100         | 100     |
    And the liquidity provisions should have the following states:
      | id  | party          | market    | commitment amount | status        |
      | lp1 | lprov1         | ETH/DEC21 | 5999              | STATUS_ACTIVE |
      | lp2 | lp2Bdistressed | ETH/DEC21 | 1                 | STATUS_ACTIVE |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | buy  | 50     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | sell | 50     | 1010  | 1                | TYPE_LIMIT | TIF_FOK |
    And the network moves ahead "5" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | open interest | target stake | supplied stake |
      | 1010       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 10            | 1010         | 5999           |
    And the liquidity provisions should have the following states:
      | id  | party          | market    | commitment amount | status           |
      | lp2 | lp2Bdistressed | ETH/DEC21 | 1                 | STATUS_CANCELLED |
