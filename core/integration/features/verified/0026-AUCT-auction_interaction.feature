Feature: Test interactions between different auction types (0035-LIQM-001)

  # Related spec files:
  #  ../spec/0026-auctions.md
  #  ../spec/0032-price-monitoring.md
  #  ../spec/0035-liquidity-monitoring.md

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0     |
      | network.floatingPointUpdates.delay            | 10s   |
      | market.auction.minimumDuration                | 10    |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99        | 300               |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 2       | 0.999995    | 200               |
      | 1       | 0.999999    | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1     | default-margin-calculator | 10               | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         |
      | ETH/DEC22 | ETH        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 10               | fees-config-1 | price-monitoring-2 | default-eth-for-future | 0.5                    | 0                         |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | lpprov | ETH   | 1000000000 |

  Scenario: Assure minimum auction length is respected
    Given the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 400   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 4000           | 10            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # try to end price auction (gets extended by min auction length as max(400,300) = 400)
    When the network moves ahead "350" blocks

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    When the network moves ahead "50" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario: When trying to exit opening auction liquidity monitoring doesn't get triggered, hence the opening auction uncrosses and market goes into continuous trading mode (0026-AUCT-004)

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 500        | 10     | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"

    # target_stake = mark_price x max_oi x target_stake_scaling_factor x max(risk_factor_long, risk_factor_short) = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 10000          | 10            |

  Scenario: When trying to exit opening auction liquidity monitoring is triggered due to missing best bid, hence the opening auction gets extended (0026-AUCT-005)

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | extension trigger                          |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks
    Then the auction ends with a traded volume of "10" at a price of "1000"

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 10000          | 10            |

  Scenario: When trying to exit opening auction liquidity monitoring is triggered due to insufficient supplied stake  (0026-AUCT-004, 0026-AUCT-005)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | extension trigger                        |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 799               | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 799               | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 799               | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 799               | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 1000         | 799            | 0             |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | MID              | 2          | 1      | amendment |

    And the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | extension trigger                        | target stake | supplied stake | open interest |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 1000         | 800            | 0             |

    When the network moves ahead "1" blocks

    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks

    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

  Scenario: Once market is in continuous trading mode: post a persistent order that should trigger liquidity auction (not enough target stake), appropriate event is sent and market in TRADING_MODE_MONITORING_AUCTION (0026-AUCT-005, 0035-LIQM-003)
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | submission |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    Then clear all events

    # Trade is expected to go through
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 20     | 1010  | 3                | TYPE_LIMIT | TIF_GTC |

    # verify that we don't enter liquidity auction immediately, but at the end of block as per 0035-LIQM-003
    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3000         | 1000           | 30            |

    # move to the next block to perform liquidity check
    Then the network moves ahead "1" blocks
    # open interest updates to include buy order of size 20
    And the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 3030         | 1000           | 30            |

    Then the following events should be emitted:
      | type                    |
      | PositionStateEvent      |
      | MarginLevelsEvent       |
      | AccountEvent            |
      | LedgerMovements         |
      | LiquidityProvisionEvent |
      | OrderEvent              |
      | AuctionEvent            |
      | MarketUpdatedEvent      |
    # LP repricing, checking, and cancelling emits a ton of events
    And a total of "110" events should be emitted

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "11" blocks

    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3030         | 10000          | 30            |

  Scenario: Once market is in continuous trading mode: post a non-persistent order that should trigger liquidity auction (not enough target stake), still goes through, but auction is triggered the next block
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | submission |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1012  | 1      |
      | sell | 1010  | 1      |
      | sell | 1001  | 1      |
      | buy  | 999   | 1      |
      | buy  | 990   | 1      |
      | buy  | 988   | 1      |
      | buy  | 900   | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC21 | sell | 2      | 1010  | 0                | TYPE_LIMIT | TIF_GFN | ref-ref   |
      | party1 | ETH/DEC21 | buy  | 20     | 1010  | 3                | TYPE_LIMIT | TIF_GFN | ref-ref   |

    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1400         | 1000           | 14            |
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 1414         | 1000           | 14            |

  Scenario: Once market is in continuous trading mode: post a non-persistent order that should trigger liquidity auction (no best ask), the order trades, market goes into auction mode and an appropriate event is sent and market goes into TRADING_MODE_MONITORING_AUCTION the next block (0035-LIQM-002)
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 991   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1009  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 100          | 2000           | 1             |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1011  | 1      |
      | sell | 1009  | 1      |
      | sell | 1001  | 2      |
      | buy  | 999   | 2      |
      | buy  | 991   | 1      |
      | buy  | 989   | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party1 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_MARKET | TIF_IOC |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1011  | 0      |
      | sell | 1009  | 0      |
      | sell | 1001  | 0      |
      | buy  | 999   | 0      |
      | buy  | 991   | 1      |
      | buy  | 989   | 0      |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | 1009              | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 400          | 2000           | 4             |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode                    | auction trigger                            | target stake | supplied stake | open interest |
      | 1009       | 1009              | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS | 403          | 2000           | 4             |

  Scenario: Once market is in continuous trading mode: post a non-persistent order that should trigger liquidity auction (no best ask), the order trades, then provide more orders that ensure a best ask price exists before the end of the block. The market should not enter auction. Same scenario as above until the last "network moves ahead 1 blocks" bit)
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 991   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1009  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 100          | 2000           | 1             |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1010  | 1      |
      | sell | 1009  | 1      |
      | sell | 1001  | 2      |
      | buy  | 999   | 2      |
      | buy  | 991   | 1      |
      | buy  | 990   | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party1 | ETH/DEC21 | buy  | 4      | 1011  | 3                | TYPE_MARKET | TIF_IOC |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 1      | 0              | 0            |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | 1010              | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 500          | 2000           | 5             |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1010  | 0      |
      | sell | 1009  | 0      |
      | sell | 1001  | 0      |
      | buy  | 999   | 0      |
      | buy  | 991   | 1      |
      | buy  | 990   | 0      |


    # replenish best ask
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | sell | 1      | 1009  | 0                | TYPE_LIMIT | TIF_GTC |

    #move forwards to next block (check LP)
    And the network moves ahead "1" blocks
    #we should still be in continuous trading
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 100     | 995       | 1014      | 505          | 2000           | 5             |
    # orderbook should have the same state as before the static sell side got fully consumed
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1010  | 1      |
      | sell | 1009  | 1      |
      | sell | 1001  | 2      |
      | buy  | 999   | 2      |
      | buy  | 991   | 1      |
      | buy  | 990   | 1      |

  Scenario: Once market is in continuous trading mode: post a non-persistent order that should trigger price auction, check that the order gets stopped, appropriate event is sent and market remains in TRADING_MODE_CONTINUOUS, 0024-OSTA-012
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | submission |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                       |
      | party2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | no-reject |                                                             |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GFN | reject-me | OrderError: non-persistent order trades out of price bounds |
    Then the following orders should be stopped:
      | party  | market id | reason                                               |
      | party1 | ETH/DEC21 | ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                       |
      | party2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC | no-reject |                                                             |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_FOK | reject-me | OrderError: non-persistent order trades out of price bounds |
    Then the following orders should be stopped:
      | party  | market id | reason                                               |
      | party1 | ETH/DEC21 | ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS |

    Then the network moves ahead "5" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> leave auction mode (0068-MATC-033,0026-AUCT-005)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | submission |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1012  | 1      |
      | sell | 1010  | 1      |
      | sell | 1001  | 1      |
      | buy  | 999   | 1      |
      | buy  | 988   | 1      |
      | buy  | 900   | 1      |

    # If the order traded there'd be insufficient liquidity for the market to operate, the next block will trigger auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 3      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 4      | 1010  | 3                | TYPE_LIMIT | TIF_FOK |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | 1010              | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1400         | 1000           | 14            |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | horizon | min bound | max bound |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 993       | 1012      |

    # submit the order during auction so we can cancel it later on
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    When the network moves ahead "10" blocks
    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties cancel the following orders:
      | party  | reference   |
      | party1 | cancel-me-1 |
      | party2 | cancel-me-2 |

    When the network moves ahead "2" blocks

    Then  the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | MID              | 2          | 1      | amendment |

    # leave liquidity auction
    When the network moves ahead "2" blocks

    # We should be able to leave liquidity auction now (price extension keep the market in auction mode though)
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | extension trigger     |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_PRICE |

    # End price auction extension
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 100     | 1010      | 1030      | 3468         | 4080           | 34            |

  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> extend with liquidity monitoring -> leave auction mode (0068-MATC-033,0026-AUCT-005)
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | submission |

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
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1012  | 1      |
      | sell | 1010  | 1      |
      | sell | 1001  | 1      |
      | buy  | 999   | 1      |
      | buy  | 988   | 1      |
      | buy  | 900   | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party2 | ETH/DEC21 | sell | 2      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | trigger-liq |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 3                | TYPE_LIMIT | TIF_GTC | trigger-liq |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | 1010              | TRADING_MODE_CONTINUOUS | 1400         | 1000           | 14            |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    When the network moves ahead "1" blocks
    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties cancel the following orders:
      | party  | reference   |
      | party1 | cancel-me-1 |
      | party2 | cancel-me-2 |

    When the network moves ahead "1" blocks

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 3468              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 3468              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 3468              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 3468              | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "11" blocks
    # We should be able to leave liquidity auction now (price extension keep the market in auction mode though)
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | extension trigger     |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_PRICE |

    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    # Jump ahead to the end of the price monitoring auction period
    When the network moves ahead "301" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 4488         | 3468           | 14            |

    When the network moves ahead "1" blocks

    # Increasing the supplied stake should end the auction
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 4488              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4488              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4488              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4488              | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 4488         | 4488           | 44            |

  Scenario: Once market is in continuous trading mode: enter price monitoring auction -> extend with liquidity monitoring auction -> leave auction mode (0068-MATC-033,0026-AUCT-005)
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 2000              | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 2000           | 10            |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # end price auction (gets extended by liquidity)
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       | extension trigger                        |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    # We were in liquidity auction, we've updated the commitment amount
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4000              | 0.001 | sell | MID              | 2          | 1      | amendment |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | 100     | 1010      | 1030      | 3060         | 4000           | 30            |

  Scenario: Once market is in continuous trading mode: enter liquidity monitoring auction -> extend with price monitoring auction -> extend with liquidity auction -> leave auction mode (0068-MATC-033, 0026-AUCT-005)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.8              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | updated-lqm-params   | 0.5                    | 0                         |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 100     | 990       | 1010      | 1000         | 1000           | 10            |

    # Triggering liquidity monitoring auction
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | trigger-l-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 1                | TYPE_LIMIT | TIF_GTC | trigger-l-2 |
    Then the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | extension trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_UNSPECIFIED |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-1 |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | cancel-me-2 |

    And the network moves ahead "1" blocks
    Then the parties cancel the following orders:
      | party  | reference   |
      | party1 | cancel-me-1 |
      | party2 | cancel-me-2 |

    # Updating the commitment amount to come out of liquidity auction
    And  the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 2080              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 2080              | 0.001 | sell | ASK              | 1          | 2      | amendment |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_UNSPECIFIED | 100     | 995       | 1015      | 2020         | 2080           | 20            |

    # Now we place some orders that are outside the price range to trigger price auction
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
    And  the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 4080              | 0.001 | sell | ASK              | 1          | 2      | amendment |

    Then the network moves ahead "9" blocks

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger     | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_PRICE | 4080         | 4080           | 20            |
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          | extension trigger     |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_PRICE |

    # increase open interest
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
    # Jump ahead to when the end price monitoring auction should end
    And the network moves ahead "301" blocks

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger                        | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 6120         | 4080           | 20            |

    When the network moves ahead "1" blocks

    # Updating the commitment amount to come out of liquidity auction
    Then  the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 6120              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 6120              | 0.001 | sell | ASK              | 1          | 2      | amendment |

    When the network moves ahead "10" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1020       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 100     | 1010      | 1030      | 6120         | 6120           | 60            |
