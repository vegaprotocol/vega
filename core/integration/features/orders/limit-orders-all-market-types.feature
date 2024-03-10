Feature: limit orders in all market types

  # We try to exercise limit orders as much as possible so that we can check
  # they work the same in both futures and perpetual markets
 
  # All order types should be able to be placed and act in the same way on a perpetual
  # market as on an expiring future market. Specifically this includes: Limit orders (0014-ORDT-120)

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC23 | ETH        | ETH   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  Scenario:1 In continuous trading we can submit orders with all tif options in the same block
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | party4 | ETH   | 10000000 |
      | lp1    | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
   When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_IOC |        | ioc order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_FOK |        | fok order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GFN |        | gfn order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_IOC |        | ioc order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_FOK |        | fok order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GFN |        | gfn order received during auction trading |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Auction only orders should be rejected
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GFA |        | gfa order received during continuous trading |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GFA |        | gfa order received during continuous trading |


    # A non matching GTC should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | post   |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | post   |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      

    # A matching GTC should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |

    # A non matching GTT should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | expires in | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |

    # A matching GTT should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | expires in | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |

    # A non matching IOC should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | post    |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | post    |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |

    # A matching IOC should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |

    # A non matching FOK should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | post    |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | post    |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |

    # A matching FOK should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |



  Scenario:2 In continuous trading we can submit orders with all tif options with each order in a different block
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | party4 | ETH   | 10000000 |
      | lp1    | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
   When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_IOC |        | ioc order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_FOK |        | fok order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GFN |        | gfn order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_IOC |        | ioc order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_FOK |        | fok order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GFN |        | gfn order received during auction trading |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Auction only orders should be rejected
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GFA |        | gfa order received during continuous trading |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GFA |        | gfa order received during continuous trading |


    # A non matching GTC should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | post   |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | post   |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      

    # A matching GTC should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTC |        |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |

    # A non matching GTT should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | expires in | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |

    # A matching GTT should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | expires in | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTT | 50         |        |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | post   | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |

    # A non matching IOC should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | post    |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | post    |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |

    # A matching IOC should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_IOC |         |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |

    # A non matching FOK should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | post    |       |
      | party1| ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | post    |       |
      | party1| ETH/DEC23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |

    # A matching FOK should be accepted
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 1                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_FOK |         |       |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |

