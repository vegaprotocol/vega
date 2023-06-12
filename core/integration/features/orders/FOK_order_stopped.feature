Feature: check when FOK market order unable to trade

  Background:
    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the margin calculator named "margin-calculator-0":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-0 | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
     
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
  And the average block duration is "1"

Scenario: 001 FOK market order unable to trade
    Given the parties deposit on asset's general account the following amount:
      | party       | asset | amount        |
      | auxiliary1  | USD   | 1000000000000 |
      | auxiliary2  | USD   | 1000000000000 |
      | trader2     | USD   | 90000         |
      | trader20    | USD   | 10000         |
      | trader3     | USD   | 90000         |
      | lprov       | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp0 | lprov | ETH/DEC20 | 5000              | 0.001 | sell | ASK              | 100        | 55     | submission |
      | lp0 | lprov | ETH/DEC20 | 5000              | 0.001 | buy  | BID              | 100        | 55     | submission  |

    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-50    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10000 |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-10    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10    |

    When the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "10" for the market "ETH/DEC20"
    Then debug orders
    And the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | buy  | 1     | 5000   |
      | buy  | 5     | 5      |
      | sell | 1000  | 10     |
      | sell | 1005  | 5      |

    # setup trader2 position for an order which is partially filled and leading to a reduced position
    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | auxiliary1 | ETH/DEC20 | buy  | 1000   | 0     | 0                | TYPE_MARKET | TIF_FOK| FOK-order-market |

    And the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | buy  | 1     | 5000   |
      | buy  | 5     | 5      |
      | sell | 1000  | 10     |
      | sell | 1005  | 5      |

    # check the order status, it should be stopped
    And the orders should have the following status:
      | party      | reference       | status         |
      | auxiliary1 |FOK-order-market | STATUS_STOPPED |
      
