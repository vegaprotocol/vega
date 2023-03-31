Feature: Replicate issue 3528, where price monitoring continuously extended liquidity auction

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0.9   |
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
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.01                   | 0                         |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 100000000  |
      | party5 | ETH   | 100000000  |
      | party6 | ETH   | 100000000  |
      | party7 | ETH   | 100000000  |

  Scenario: Enter liquidity auction, extended by trades at liq. auction end, single trade -> single extension

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | extension trigger                        |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |
    # liquidity auction should only have an end time at T+1s
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | extension trigger                        | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 990       | 1010      | 1000         | 700            | 0             | 2           |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"
    # liquidity auction is extended by 1 second this block (duration accrues)
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | extension trigger                        | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 990       | 1010      | 1000         | 800            | 0             | 3           |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | sell | MID              | 2          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | extension trigger                        |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |
    # we're still in the same block so auction end is start + 3 seconds now
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | extension trigger                        | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 990       | 1010      | 1010         | 801            | 0             | 3           |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks

    # we've met the liquidity requirements so the opening auction uncrosses now
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest | auction end |
      | 1010       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1010         | 10000          | 10            | 0           |

  Scenario: Enter liquidity auction, extended by trades at liq. auction end, multiple trades -> still a single extension

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | -2     | submission |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | -1     | amendment  |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                 | auction trigger         | extension trigger                        |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    # opening auction (extended by liquidity auction() should have an end time at T+2s
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | extension trigger                        | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 990       | 1010      | 1000         | 700            | 0             | 2           |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"
    # liquidity auction is extended by 1 second this block
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | extension trigger                        | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 990       | 1010      | 1000         | 800            | 0             | 3           |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | sell | MID              | 2          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    # Still in the same block, so auction end is start + 3 seconds now
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                 | auction trigger         | extension trigger                        | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 100     | 990       | 1010      | 1010         | 801            | 0             | 3           |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 1010              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 1010              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 1010              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 1010              | 0.001 | sell | MID              | 2          | 1      | amendment |

    When the network moves ahead "1" blocks

    # liquidity requirements met -> opening auction finishes (price monitoring extension not possible in opening auction)
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest | auction end | min bound | max bound |
      | 1010       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1010         | 1010           | 10            | 0           | 1001      | 1019      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      #| party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | This trade does happen now that we've changed liquidity checks
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 2                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks

    # open interest changes from 10 to 20, because the trade _does_ happen
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger           | target stake | supplied stake | open interest | auction end | min bound | max bound |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_UNSPECIFIED | 2020         | 1010           | 20            | 1           | 1001      | 1019      |
    #| 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_UNSPECIFIED | 2020         | 1010           | 10            | 1           |  1001      | 1019     |

    # Place order outwith price monitoring bounds
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 5      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC21 | buy  | 3      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/DEC21 | buy  | 2      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 6      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/DEC21 | sell | 4      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks

    # trade at 1010 changes the target stake, too
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger                        | target stake | supplied stake | open interest | auction end | min bound | max bound |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 3090         | 1010           | 20            | 2           | 1001      | 1019      |
    #| 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 2050         | 1010           | 20            | 2           |  1001      | 1019     |

    When the network moves ahead "10" blocks

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment |

    # we've met the liquidity requirements, but the auction uncrosses out of bounds
    # Auction end is now 12s (2+10 blocks) + 300 price extension
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger     | target stake | supplied stake | open interest | auction end |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_PRICE | 3090         | 10000          | 20            | 312         |

    When the network moves ahead "150" blocks

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | extension trigger     | target stake | supplied stake | open interest | auction end |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | AUCTION_TRIGGER_PRICE | 3090         | 10000          | 20            | 312         |

    Then the network moves ahead "150" blocks

    # price auction ends as expected event though uncrossing price is outwith the previous bounds (price extension can be called at most once per trigger)
    # Now the open interest is 30 (previously was 20) -> because the initial trade at 1010 went through. The target stake is increased because of the time + leaving auction
    # The price bounds have also changed from 1016-1034, and the mark price is now 1030 (the 1010 orders are gone, so the uncrossing price is different)
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1030       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 100     | 1020      | 1040      | 3090         | 10000          | 30            | 0           |
