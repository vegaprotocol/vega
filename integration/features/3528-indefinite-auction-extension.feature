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
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
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
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |
    And the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | trader0 | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |
      | trader3 | ETH   | 100000000  |
      | trader4 | ETH   | 100000000  |
      | trader5 | ETH   | 100000000  |
      | trader6 | ETH   | 100000000  |
      | trader7 | ETH   | 100000000  |

  Scenario: Enter liquidity auction, extended by trades at liq. auction end, single trade -> single extension

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
  # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
  # liquidity auction should only have an end time at T+1s
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 700            | 10            | 1           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"
  # liquidity auction is extended by 1 second this block
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 800            | 10            | 2           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | sell | MID              | 2          | 1      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
  # liquidity auction is extended by 1 second this block, so auction end is start + 3 seconds now
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 801            | 10            | 2           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks

  # we've met the liquidity requirements, but the auction uncrosses out of bounds
  # Auction end is now 4s (4 blocks) + 300 price extension
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 10000          | 10            | 302         |
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 10     | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1040  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "10" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1030      | 1000         | 10000          | 10            | 302         |

  # place some more orders out of bounds
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 10     | 1041  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1039  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "290" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1030       | TRADING_MODE_CONTINUOUS | 1       | 1020      | 1040      | 3090         | 10000          | 30            | 0           |

  Scenario: Enter liquidity auction, extended by trades at liq. auction end, multiple trades -> still a single extension

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
  # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
  # liquidity auction should only have an end time at T+1s
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 700            | 10            | 1           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 800               | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"
  # liquidity auction is extended by 1 second this block
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 800            | 10            | 2           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 801               | 0.001 | sell | MID              | 2          | 1      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 10     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
  # liquidity auction is extended by 1 second this block, so auction end is start + 3 seconds now
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 801            | 10            | 2           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks

  # we've met the liquidity requirements, but the auction uncrosses out of bounds
  # Auction end is now 4s (4 blocks) + 300 price extension
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 10000          | 10            | 302         |
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 5      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader5 | ETH/DEC21 | buy  | 2      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC21 | sell | 6      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader6 | ETH/DEC21 | sell | 4      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "10" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 10000          | 10            | 302         |

  # place some more orders out of bounds
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 5      | 1041  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1041  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader5 | ETH/DEC21 | buy  | 2      | 1041  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1039  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC21 | sell | 6      | 1039  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader6 | ETH/DEC21 | sell | 4      | 1039  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "290" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1030       | TRADING_MODE_CONTINUOUS | 1       | 1020      | 1040      | 3090         | 10000          | 30            | 0           |

  Scenario: When in liquidity auction, we should only trigger price extension once

    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8 |
    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
  # In this case, the required time has expired, and the book is fine, so the trigger probably should be LIQUIDITY
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
  # liquidity auction should only have an end time at T+1s
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 700            | 10            | 1           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 700               | 0.001 | sell | MID              | 2          | 1      |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 5      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader5 | ETH/DEC21 | buy  | 2      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC21 | sell | 6      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader6 | ETH/DEC21 | sell | 4      | 1040  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"
  # liquidity auction is extended by 1 second this block
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 700            | 10            | 2           |

    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"
  # liquidity auction is extended by 1 second this block
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 990       | 1010      | 1000         | 700            | 10            | 3           |

    And the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -2     |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | -1     |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      |
      | lp1 | trader0 | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 10000          | 10            | 303         |

  # We've extended the auction through price monitoring
    When the network moves ahead "150" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | target stake | supplied stake | open interest | auction end |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1000         | 10000          | 10            | 303         |
  # price auction ends as expected
    When the network moves ahead "150" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1040       | TRADING_MODE_CONTINUOUS | 1       | 1030      | 1050      | 2080         | 10000          | 20            | 0           |
