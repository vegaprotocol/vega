Feature: Replicate issue 3528, where price monitoring continuously extended liquidity auction

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | limits.markets.maxPeggedOrders                | 4     |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.9              | 24h         | 1              |  
    
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
      | id        | quote name | asset | liquidity monitoring | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH        | ETH   | lqm-params           | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.01                   | 0                         | default-futures |
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
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1          | 2      |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | MID              | 2          | 1      |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1          | 2      |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | MID              | 2          | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | amendment |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    # liquidity auction is extended by 1 second this block (duration accrues)

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 20     | 1020  | 1                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | amendment |

    When the network moves ahead "1" blocks

    ## Price auction kicked off
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 2244         | 10000          | 12            | 300         |
    ## End of price auction -> trade goes through and the mark price is updated
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest | auction end |
      | 1020       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 2244         | 10000          | 22            | 0           |


  Scenario: Enter liquidity auction, extended by trades at liq. auction end, multiple trades -> still a single extension

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/DEC21 | 700               | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 2      |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | MID              | 2      | 1      |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 2      |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | MID              | 2      | 1      |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    # opening auction (extended by liquidity auction() should have an end time at T+2s
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 800               | 0.001 | amendment |

    When the network moves ahead "1" blocks

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 801               | 0.001 | amendment |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1020  | 1                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 1010              | 0.001 | amendment |

    When the network moves ahead "1" blocks

    # liquidity requirements met -> opening auction finishes (price monitoring extension not possible in opening auction)
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest | auction end | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 2040         | 1010           | 12            | 300         | 1001      | 1019      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks


    # Place order outwith price monitoring bounds
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 5      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC21 | buy  | 3      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/DEC21 | buy  | 2      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 6      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/DEC21 | sell | 4      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "10" blocks

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type   |
      | lp1 | party0 | ETH/DEC21 | 10000             | 0.001 | amendment |

    # Auction end is now 290 blocks away

    When the network moves ahead "290" blocks

    # price auction ends as expected event though uncrossing price is outwith the previous bounds (price extension can be called at most once per trigger)
    # Now the open interest is 30 (previously was 20) -> because the initial trade at 1010 went through. The target stake is increased because of the time + leaving auction
    # The price bounds have also changed from 1016-1034, and the mark price is now 1030 (the 1010 orders are gone, so the uncrossing price is different)
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1020       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 100     | 1010      | 1030      | 3060         | 10000          | 30            | 0           |
