Feature: Set up a market with an opening auction manipulate price to low level with low collateral, move price to higher level, manipulator orders should get cancelled

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 10               | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party        | asset | amount    |
      | manipulator1 | BTC   | 3         |
      | manipulator2 | BTC   | 3         |
      | party3       | BTC   | 100000000 |
      | party4       | BTC   | 100000000 |
      | party5       | BTC   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | limits.markets.maxPeggedOrders          | 2     |
      | network.markPriceUpdateMaximumFrequency | 1s    |
  
  Scenario: 
    When the parties place the following orders with ticks:
      | party         | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | manipulator1  | ETH/DEC19 | buy  | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | mani-1    |
      | manipulator2  | ETH/DEC19 | sell | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | mani-2    |
    Then the parties should have the following margin levels:
      | party        | market id | maintenance |
      | manipulator1 | ETH/DEC19 | 2           |
      | manipulator2 | ETH/DEC19 | 2           |
    And the orders should have the following states:
      | party        | market id | reference | side | volume | remaining | price | status        |
      | manipulator1 | ETH/DEC19 | mani-1    | buy  | 10     | 10        | 2     | STATUS_ACTIVE |
      | manipulator2 | ETH/DEC19 | mani-2    | sell | 10     | 10        | 2     | STATUS_ACTIVE |
    And the market data for the market "ETH/DEC19" should be:
      | trading mode                 | open interest | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0             | 2                | 10                |


    When the parties place the following orders with ticks:
      | party         | market id | side | volume | price | resulting trades | type       | tif     |
      | party3        | ETH/DEC19 | buy  | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4        | ETH/DEC19 | sell | 20     | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following margin levels:
      | party        | market id | maintenance |
      | manipulator1 | ETH/DEC19 | 2           |
      | manipulator2 | ETH/DEC19 | 2           |
      | party3       | ETH/DEC19 | 200         |
      | party4       | ETH/DEC19 | 200         |
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 | open interest | indicative price | indicative volume |
      | TRADING_MODE_OPENING_AUCTION | 0             | 100              | 20                |

    # 0019-MCAL-234
    When the parties place the following orders with ticks:
      | party         | market id | side | volume | price | resulting trades | type       | tif     |
      | party5        | ETH/DEC19 | buy  | 10     | 3     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following margin levels:
      | party        | market id | maintenance |
      | party5       | ETH/DEC19 | 100         |

    When the network moves ahead "5" blocks
    Then the parties should have the following margin levels:
      | party        | market id | maintenance |
      | manipulator1 | ETH/DEC19 | 2           |
      | manipulator2 | ETH/DEC19 | 2           |
      | party3       | ETH/DEC19 | 200         |
      | party4       | ETH/DEC19 | 200         |
    And the orders should have the following states:
      | party        | market id | reference | side | volume | remaining | price | status        |
      | manipulator1 | ETH/DEC19 | mani-1    | buy  | 10     | 10        | 2     | STATUS_ACTIVE |
      | manipulator2 | ETH/DEC19 | mani-2    | sell | 10     | 10        | 2     | STATUS_ACTIVE |

    When the network moves ahead "11" blocks  
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode            | open interest |
      | TRADING_MODE_CONTINUOUS | 20            |
    Then debug orders
    And the orders should have the following states:
      | party        | market id | reference | side | volume | remaining | price | status         |
      | manipulator1 | ETH/DEC19 | mani-1    | buy  | 10     | 10        | 2     | STATUS_STOPPED |
      | manipulator2 | ETH/DEC19 | mani-2    | sell | 10     | 0         | 2     | STATUS_FILLED  |
