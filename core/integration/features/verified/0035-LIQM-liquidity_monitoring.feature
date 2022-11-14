Feature: Test liquidity monitoring

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 1s   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 1     |
      | network.floatingPointUpdates.delay            | 5s    |
      | market.auction.minimumDuration                | 1     |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
    And the average block duration is "2"
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
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1     | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | lprov1 | ETH   | 1000000000 |
      | lprov2 | ETH   | 1000000000 |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | lprov1 | USD   | 500000    |
      | lprov2 | USD   | 500000    |

  Scenario: A market which enters a state requiring liquidity auction through increased open interest during a block but then leaves state again prior to block completion never enters liquidity auction. (0035-LIQM-005)
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 1000           | 10            |

    Then clear all events

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 20     | 1010  | 2                | TYPE_LIMIT | TIF_GTC |
    
    # verify that we don't enter liquidity auction immediately despite liquidity being undersuplied
    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3030         | 1000           | 30            |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lprov1 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lprov1 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment |

    # move to the next block to perform liquidity check, we should still be in continuous trading
    Then the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3030         | 10000          | 30            |

    # verify that at no point auction has been entered
    Then the following events should NOT be emitted:
      | type                               |
      | AuctionEvent                       |   

Scenario: A market which enters a state requiring liquidity auction through reduced current stake (e.g. through LP bankruptcy) during a block but then leaves state again prior to block completion never enters liquidity auction. (0035-LIQM-006)
   Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.24  |

   And the average block duration is "1"

   And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov1 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | lprov1 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

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
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 213414       | 50000          | 60            |
    And clear all events
    
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lprov1 | ETH/MAR22 | sell | 15     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 213414       | 50000          | 60            |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lprov2 | ETH/MAR22 | 1220              | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp2 | lprov2 | ETH/MAR22 | 1220              | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    # move to the next block to perform liquidity check, we should still be in continuous trading
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/MAR22" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 213414       | 51220          | 60            |

    # verify that at no point auction has been entered
    Then the following events should NOT be emitted:
      | type                               |
      | AuctionEvent                       |   

Scenario: A liquidity provider cannot remove their liquidity within the block if this would bring the current total stake below the target stake as of that transaction. (0035-LIQM-007)
  Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 1     |
      | market.stake.target.timeWindow                | 1s    |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lprov1 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lprov2 | ETH/DEC21 | 500               | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lprov2 | ETH/DEC21 | 500               | 0.001 | sell | ASK              | 1          | 2      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 15     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 15     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "15" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1500         | 1500           | 15            |

    Then the network moves ahead "1" blocks

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    | error                                            |
      | lp2 | lprov2 | ETH/DEC21 | 400               | 0.001 | buy  | BID              | 1          | 2      | amendment  | commitment submission rejected, not enough stake |
      | lp2 | lprov2 | ETH/DEC21 | 400               | 0.001 | sell | ASK              | 1          | 2      | amendment  |                                                  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1300         | 1500           | 13            |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type      |  error                                            |
      | lp2 | lprov2 | ETH/DEC21 | 0                 | 0.001 | buy  | BID              | 1          | 2      | cancellation |  commitment submission rejected, not enough stake |
      | lp2 | lprov2 | ETH/DEC21 | 0                 | 0.001 | sell | ASK              | 1          | 2      | cancellation |                                                   |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | sell | 3      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | buy  | 3      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 1500           | 10            |

    # cancellation goes through fine once target stake's been updated 
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type      |
      | lp2 | lprov2 | ETH/DEC21 | 0                 | 0.001 | buy  | BID              | 1          | 2      | cancellation |
      | lp2 | lprov2 | ETH/DEC21 | 0                 | 0.001 | sell | ASK              | 1          | 2      | cancellation |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 1000           | 10            |

  Scenario:  When the Max Open Interest field decreases for a created block to a level such that a liquidity auction which is active at the start of a block can now be exited the block stays in auction within the block but leaves at the end. (0035-LIQM-008)
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 1     |
      | market.stake.target.timeWindow                | 5s    |
    And the parties deposit on asset's general account the following amount:  
      | party  | asset | amount     |
      | lp2Bdistressed | ETH   | 6400       |
    And the parties submit the following liquidity provision:
      | id  | party          | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp2Bdistressed | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | lp2Bdistressed | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |
      | lp2 | lprov1         | ETH/DEC21 | 1011              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lprov1         | ETH/DEC21 | 1011              | 0.001 | sell | ASK              | 1          | 2      | amendment |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | lprov1 | ETH/DEC21 | buy  | 100    | 970   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lprov1 | ETH/DEC21 | sell | 100    | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "60" at a price of "1000"
    
    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2 | ETH/DEC21 | buy  | 50     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | 
      | party1 | ETH/DEC21 | sell | 53     | 1010  | 1                | TYPE_LIMIT | TIF_GTC | 
    Then the liquidity provisions should have the following states:
      | id  | party          | market    | commitment amount | status           |
      | lp1 | lp2Bdistressed | ETH/DEC21 | 5000              | STATUS_CANCELLED |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 6060         | 1011           | 10            |

    # target stake should drop now as the OI of 60 drops outside the window over the course of the 2nd block
    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger           | target stake | supplied stake  | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1010         | 1011            | 10            |
