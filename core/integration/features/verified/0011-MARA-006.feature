Feature: check margin account with partially filled order

  Background:
    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the margin calculator named "margin-calculator-0":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 300               |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring  | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-0 | 1                | default-none | default-none      | default-eth-for-future | 20                   | 0                         | default-futures |
      | ETH/DEC21 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-0 | 1                | default-none | price-monitoring-1| default-eth-for-future | 20                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
  And the average block duration is "1"

  @Liquidation
  Scenario: 001 If an order is partially filled and if this leads to a reduced position and reduced riskiest long / short then the margin requirements are seen to be reduced and if margin balance is above release level then the excess amount is transferred to the general account.0011-MARA-006. 0011-MARA-008
    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 90000         |
      | trader20   | USD   | 1000000000000 |
      | trader3    | USD   | 90000         |
      | lprov      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp0 | lprov | ETH/DEC20 | 100000            | 0.001 | submission |
      | lp0 | lprov | ETH/DEC20 | 100000            | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lprov | ETH/DEC20 | 100 | 1 | sell | ASK | 100 | 55 |
      | lprov | ETH/DEC20 | 100 | 1 | buy  | BID | 100 | 55 |

    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-50    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10000 |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-10    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10    |

    When the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "10" for the market "ETH/DEC20"

    # setup trader2 position for an order which is partially filled and leading to a reduced position
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader2  | ETH/DEC20 | sell | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-3 |
      | trader20 | ETH/DEC20 | buy  | 40     | 50    | 1                | TYPE_LIMIT | TIF_GTC | buy-order-3 |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 47114       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader2 | ETH/DEC20 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTC | buy-order-4 |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 47114       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | trader20 | ETH/DEC20 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTC | sell-order-4 |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 35336       |

    # margin is under above  level, then the excess amount is transferred to the general account
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD | ETH/DEC20   | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | trader20 | ETH/DEC20 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC | sell-order-4 |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 34158       |

    # margin is under release level, then no excess amount is transferred to the general account
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 51237  | 38763   |

@Liquidation
Scenario: 002 check margin for GTT order type.0011-MARA-007
    Given the parties deposit on asset's general account the following amount:
      | party       | asset | amount        |
      | auxiliary1  | USD   | 1000000000000 |
      | auxiliary2  | USD   | 1000000000000 |
      | trader2     | USD   | 90000         |
      | trader20    | USD   | 1000000000000 |
      | trader3     | USD   | 90000         |
      | trader4     | USD   | 90000         |
      | lprov       | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp0 | lprov | ETH/DEC20 | 100000            | 0.001 | submission |
      | lp0 | lprov | ETH/DEC20 | 100000            | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 100    | 55     |
      | lprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 100    | 55     |

    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |   6        |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |   6        |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC |   6        |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC |   6        |

    When the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "10" for the market "ETH/DEC20"

    # setup trader2 position for an order which is partially filled and leading to a reduced position
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader2  | ETH/DEC20 | sell | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTT | 6          |
      | trader20 | ETH/DEC20 | buy  | 40     | 50    | 1                | TYPE_LIMIT | TIF_GTT | 6          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 47114       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |expires in |
      | trader2 | ETH/DEC20 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GTT |   6       |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 47114       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader20 | ETH/DEC20 | sell | 10     | 50    | 1                | TYPE_LIMIT | TIF_GTT | 6          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 35336       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader20 | ETH/DEC20 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTT | 6          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC20 | 34158       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC20 | 51237  | 38763   |
    # trader3 places a new order
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader3 | ETH/DEC20 | buy  | 20     | 45    | 0                | TYPE_LIMIT | TIF_GTT | 3          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader3 | ETH/DEC20 | 801         |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/DEC20 | 1201   | 88799   |

    Then the network moves ahead "7" blocks
    #GTT order expires
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC20 | 0           | 0      | 0       | 0       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/DEC20 | 0      | 90000   |
    #reset mark price
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader2  | ETH/DEC20 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTT | 6          |
      | trader20 | ETH/DEC20 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTT | 6          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC20 | 0           | 0      | 0       | 0       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/DEC20 | 0      | 90000   |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader3 | ETH/DEC20 | buy  | 10     | 45    | 0                | TYPE_LIMIT | TIF_GTT | 3          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC20 | 401         | 481    | 601     | 802     |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/DEC20 | 601    | 89399   |

    Then the network moves ahead "7" blocks
    #GTT order expires
     And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC20 | 0           | 0      | 0       | 0       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/DEC20 | 0      | 90000   |

    # now we create a case when trader 4 place a GTC order first and then GTT order
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | trader4 | ETH/DEC20 | buy  | 5      | 45    | 0                | TYPE_LIMIT | TIF_GTC |            |
      | trader4 | ETH/DEC20 | buy  | 10     | 45    | 0                | TYPE_LIMIT | TIF_GTT | 3          |

    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC20 | 601         | 721    | 901     | 1202    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader4 | USD   | ETH/DEC20 | 901    | 89099   |

    Then the network moves ahead "7" blocks
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC20 | 601         | 721    | 901     | 1202    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader4 | USD   | ETH/DEC20 | 901    | 89099   |

  @Liquidation
  Scenario: 003 check margin for GFN order type 0011-MARA-009
    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount        |
      | auxiliary1 | USD   | 1000000000000 |
      | auxiliary2 | USD   | 1000000000000 |
      | trader2    | USD   | 90000         |
      | trader20   | USD   | 9000000000000 |
      | trader3    | USD   | 90000         |
      | lprov      | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp0 | lprov | ETH/DEC21 | 100000            | 0.001 | submission |
      | lp0 | lprov | ETH/DEC21 | 100000            | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lprov | ETH/DEC21 | 2         | 1                    | sell | ASK              | 100        | 55     |
      | lprov | ETH/DEC21 | 2         | 1                    | buy  | BID              | 100        | 55     |

    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary2 | ETH/DEC21 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary1 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary2 | ETH/DEC21 | buy  | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary1 | ETH/DEC21 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 50         | TRADING_MODE_CONTINUOUS | 3600    | 49        | 51        | 17784        | 100000         | 10            |
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader2  | ETH/DEC21 | sell | 40     | 50    | 0                | TYPE_LIMIT | TIF_GFN |
      | trader20 | ETH/DEC21 | buy  | 40     | 50    | 1                | TYPE_LIMIT | TIF_GFN |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC21 | 47114       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC21 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |reference    |
      | trader2 | ETH/DEC21 | buy  | 40     | 50    | 0                | TYPE_LIMIT | TIF_GFN | GFN-order-1 |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC21 | 47114       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC21 | 70671  | 19329   |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader20 | ETH/DEC21 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GFN | t20-ref   |

    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | trader2 | ETH/DEC21 | 45936       |

# margin is under above  level, then the excess amount is transferred to the general account
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader2 | USD   | ETH/DEC21 | 70671  | 19329   |

    And the orders should have the following status:
      | party   | reference   | status        |
      | trader2 | GFN-order-1 | STATUS_ACTIVE |

    # trigger auction
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary1 | ETH/DEC21 | sell | 1      | 52    | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary2 | ETH/DEC21 | buy  | 1      | 52    | 0                | TYPE_LIMIT | TIF_GTC |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"

    #GFN order is canceled during auction
    And the orders should have the following status:
      | party   | reference   | status           |
      | trader2 | GFN-order-1 | STATUS_CANCELLED |
