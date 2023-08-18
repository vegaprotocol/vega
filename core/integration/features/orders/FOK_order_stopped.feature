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
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC20 | BTC        | USD   | log-normal-risk-model-1 | margin-calculator-0 | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
     
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 4     |
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
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp0 | lprov | ETH/DEC20 | 5000              | 0.001 | submission |
      | lp0 | lprov | ETH/DEC20 | 5000              | 0.001 | submission  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | lprov  | ETH/DEC20 | 5         | 1                    | sell | ASK              | 5000    | 5      |
      | lprov  | ETH/DEC20 | 5000      | 1                    | buy  | BID              | 5000    | 4      |

    Then the parties place the following orders:
      | party      | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | auxiliary2 | ETH/DEC20 | buy  | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-50    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10000 |
      | auxiliary2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-10    |
      | auxiliary1 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10    |

    When the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "10" for the market "ETH/DEC20"
    
    And the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | buy  | 1     | 5000   |
      | buy  | 5     | 5      |
      | sell | 1000  | 10     |
      | sell | 1005  | 5      |

    When the parties place the following orders with ticks:
      | party      | market id | side | volume | price | resulting trades | type        | tif     | reference        |
      | auxiliary1 | ETH/DEC20 | buy  | 1000   | 0     | 0                | TYPE_MARKET | TIF_FOK | FOK-order-market |

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
      
Scenario: 002 GTC and GTT order cancellaiton 
    Given the parties deposit on asset's general account the following amount:
      | party       | asset | amount        |
      | trader1     | USD   | 1000000       |
      | trader2     | USD   | 90000         |
      | trader3     | USD   | 90000         |
      | lprov       | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp0 | lprov | ETH/DEC20 | 5000              | 0.001 | submission |
      | lp0 | lprov | ETH/DEC20 | 5000              | 0.001 | submission  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | lprov  | ETH/DEC20 | 5         | 1                    | sell | ASK              | 1125    | 55     |
      | lprov  | ETH/DEC20 | 5         | 1                    | buy  | BID              | 1125    | 8      |

    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | trader2 | ETH/DEC20 | buy  | 5      | 9     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-50    |
      | trader2 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10000 |
      | trader2 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-10    |
      | trader3 | ETH/DEC20 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-10    |

    When the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "10" for the market "ETH/DEC20"
    
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | reference         |
      | trader1 | ETH/DEC20 | buy  | 1000   | 8     | 0                | TYPE_LIMIT | TIF_GTC  | GTC-order-limit-8 |
      | trader1 | ETH/DEC20 | buy  | 1000   | 7     | 0                | TYPE_LIMIT | TIF_GTC  | GTC-order-limit-7 |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general      | 
      | trader1 | USD   | ETH/DEC20 | 24022  | 975978       | 
      | lprov   | USD   | ETH/DEC20 | 60024  | 999999934976 |

    # test LP reduce commitment, margin release 
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount| fee   | lp type    |
      | lp1 | lprov | ETH/DEC20 | 4000             | 0.001 | amendment  |
      | lp1 | lprov | ETH/DEC20 | 4000             | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lprov  | ETH/DEC20 | 2         | 1                    | sell | ASK              | 100        | 55     |
      | lprov  | ETH/DEC20 | 2         | 1                    | buy  | BID              | 100        | 55     |
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general  | 
      | trader1 | USD   | ETH/DEC20 | 24022  | 975978   | 

    # test GTC order cancelation
    And the parties cancel the following orders:
      | party   | reference       |
      | trader1 | GTC-order-limit-8 |
    # check the order status, it should be stopped
    And the orders should have the following status:
      | party   | reference       | status           |
      | trader1 |GTC-order-limit-8| STATUS_CANCELLED |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general  | 
      | trader1 | USD   | ETH/DEC20 | 24022  | 975978   | 

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | reference         |
      | trader2 | ETH/DEC20 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC  | GTC-order-limit-0 |
      | trader3 | ETH/DEC20 | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC  | GTC-order-limit-1 |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general  | 
      | trader1 | USD   | ETH/DEC20 | 12012  | 987988   | 

    # test GTT order cancelation
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | reference         | expires in |
      | trader1 | ETH/DEC20 | buy  | 1000   | 8     | 0                | TYPE_LIMIT | TIF_GTT  | GTT-order-limit-8 | 2 | 

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general  | 
      | trader1 | USD   | ETH/DEC20 | 24022  | 975978   | 
  
    # test GTT order cancelation
    And the parties cancel the following orders:
      | party   | reference       |
      | trader1 | GTT-order-limit-8 |
    # check the order status, it should be stopped
    And the orders should have the following status:
      | party   | reference       | status           |
      | trader1 |GTT-order-limit-8| STATUS_CANCELLED |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general  | 
      | trader1 | USD   | ETH/DEC20 | 24022  | 975978   | 

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | reference         |
      | trader2 | ETH/DEC20 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC  | GTC-order-limit-0 |
      | trader3 | ETH/DEC20 | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC  | GTC-order-limit-1 |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general  | 
      | trader1 | USD   | ETH/DEC20 | 12012  | 987988   | 
