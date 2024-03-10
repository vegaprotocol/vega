Feature: stop orders in all market types

  # We try to exercise step orders as much as possible so that we can check
  # they work the same in both futures and perpetual markets
 
  # All order types should be able to be placed and act in the same way on a perpetual
  # market as on an expiring future market. Specifically this includes: All stop order types (0014-ORDT-123)

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

  Scenario:1 Make sure we can send buy side stop orders with all possible TIFs and order types (MARKET/LIMIT)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Stop orders require active orders or a position so create some orders here
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type        | tif     |
      | party1  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party2  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party4  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party5  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party6  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party7  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party8  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party9  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party10 | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | reduce  | 899              | 1101             | stop1-1   |       |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | post    | 899              | 1101             | stop1-2   | stop order must be reduce only |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |         | 899              | 1101             | stop1-3   | stop order must be reduce only |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | reduce  | 899              | 1101             | stop1-4   |    |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | post    | 899              | 1101             | stop1-5   | stop order must be reduce only |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |         | 899              | 1101             | stop1-6   | stop order must be reduce only |

      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce  | 899              | 1101             | stop2-1   |    |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | post    | 899              | 1101             | stop2-2   | stop order must be reduce only |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |         | 899              | 1101             | stop2-3   | stop order must be reduce only |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce  | 899              | 1101             | stop2-4   |    |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | post    | 899              | 1101             | stop2-5   | stop order must be reduce only |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |         | 899              | 1101             | stop2-6   | stop order must be reduce only |

      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | reduce  | 899              | 1101             | stop3-1   |    |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | post    | 899              | 1101             | stop3-2   | stop order must be reduce only |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN |         | 899              | 1101             | stop3-3   | stop order must be reduce only |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop3-4   |    |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | post    | 899              | 1101             | stop3-5   | stop order must be reduce only |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |         | 899              | 1101             | stop3-6   | stop order must be reduce only |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | reduce  | 899              | 1101             | stop4-1   |       |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | post    | 899              | 1101             | stop4-2   | stop order must be reduce only |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |         | 899              | 1101             | stop4-3   | stop order must be reduce only |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | reduce  | 899              | 1101             | stop4-4   |    |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | post    | 899              | 1101             | stop4-5   | stop order must be reduce only |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |         | 899              | 1101             | stop4-6   | stop order must be reduce only |

      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | reduce  | 899              | 1101             | stop5-1   |    |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | post    | 899              | 1101             | stop5-2   | stop order must be reduce only |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |         | 899              | 1101             | stop5-3   | stop order must be reduce only |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | reduce  | 899              | 1101             | stop5-4   |    |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | post    | 899              | 1101             | stop5-5   | stop order must be reduce only |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |         | 899              | 1101             | stop5-6   | stop order must be reduce only |

      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | reduce  | 899              | 1101             | stop6-1   |    |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | post    | 899              | 1101             | stop6-2   | stop order must be reduce only |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN |         | 899              | 1101             | stop6-3   | stop order must be reduce only |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-4   |    |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | post    | 899              | 1101             | stop6-5   | stop order must be reduce only |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         |         | 899              | 1101             | stop6-6   | stop order must be reduce only |





  Scenario:1b Make sure we can send buy side stop orders with all possible TIFs and order types (MARKET/LIMIT) with one order per block
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Stop orders require active orders or a position so create some orders here
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     |
      | party1  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party2  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party4  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party5  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party6  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party7  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party8  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party9  | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party10 | ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | reduce  | 899              | 1101             | stop1-1   |       |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | post    | 899              | 1101             | stop1-2   | stop order must be reduce only |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |         | 899              | 1101             | stop1-3   | stop order must be reduce only |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | reduce  | 899              | 1101             | stop1-4   |    |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | post    | 899              | 1101             | stop1-5   | stop order must be reduce only |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |         | 899              | 1101             | stop1-6   | stop order must be reduce only |

      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce  | 899              | 1101             | stop2-1   |    |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | post    | 899              | 1101             | stop2-2   | stop order must be reduce only |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |         | 899              | 1101             | stop2-3   | stop order must be reduce only |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce  | 899              | 1101             | stop2-4   |    |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | post    | 899              | 1101             | stop2-5   | stop order must be reduce only |
      | party2| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |         | 899              | 1101             | stop2-6   | stop order must be reduce only |

      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | reduce  | 899              | 1101             | stop3-1   |    |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | post    | 899              | 1101             | stop3-2   | stop order must be reduce only |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN |         | 899              | 1101             | stop3-3   | stop order must be reduce only |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop3-4   |    |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | post    | 899              | 1101             | stop3-5   | stop order must be reduce only |
      | party3| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |         | 899              | 1101             | stop3-6   | stop order must be reduce only |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | reduce  | 899              | 1101             | stop4-1   |       |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | post    | 899              | 1101             | stop4-2   | stop order must be reduce only |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |         | 899              | 1101             | stop4-3   | stop order must be reduce only |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | reduce  | 899              | 1101             | stop4-4   |    |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | post    | 899              | 1101             | stop4-5   | stop order must be reduce only |
      | party4| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |         | 899              | 1101             | stop4-6   | stop order must be reduce only |

      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | reduce  | 899              | 1101             | stop5-1   |    |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | post    | 899              | 1101             | stop5-2   | stop order must be reduce only |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |         | 899              | 1101             | stop5-3   | stop order must be reduce only |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | reduce  | 899              | 1101             | stop5-4   |    |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | post    | 899              | 1101             | stop5-5   | stop order must be reduce only |
      | party5| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |         | 899              | 1101             | stop5-6   | stop order must be reduce only |

      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | reduce  | 899              | 1101             | stop6-1   |    |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | post    | 899              | 1101             | stop6-2   | stop order must be reduce only |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN |         | 899              | 1101             | stop6-3   | stop order must be reduce only |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-4   |    |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | post    | 899              | 1101             | stop6-5   | stop order must be reduce only |
      | party6| ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         |         | 899              | 1101             | stop6-6   | stop order must be reduce only |






  Scenario:2 Make sure we can send sell side stop orders with all possible TIFs and order types (MARKET/LIMIT)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Stop orders require active orders or a position so create some orders here
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type        | tif     |
      | party1  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party2  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party4  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party5  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party6  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party7  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party8  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party9  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party10 | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | reduce  | 899              | 1101             | stop1-1   |       |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | post    | 899              | 1101             | stop1-2   | stop order must be reduce only |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |         | 899              | 1101             | stop1-3   | stop order must be reduce only |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | reduce  | 899              | 1101             | stop1-4   |    |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | post    | 899              | 1101             | stop1-5   | stop order must be reduce only |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |         | 899              | 1101             | stop1-6   | stop order must be reduce only |

      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce  | 899              | 1101             | stop2-1   |    |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | post    | 899              | 1101             | stop2-2   | stop order must be reduce only |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |         | 899              | 1101             | stop2-3   | stop order must be reduce only |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce  | 899              | 1101             | stop2-4   |    |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | post    | 899              | 1101             | stop2-5   | stop order must be reduce only |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |         | 899              | 1101             | stop2-6   | stop order must be reduce only |

      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | reduce  | 899              | 1101             | stop3-1   |    |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | post    | 899              | 1101             | stop3-2   | stop order must be reduce only |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN |         | 899              | 1101             | stop3-3   | stop order must be reduce only |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop3-4   |    |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | post    | 899              | 1101             | stop3-5   | stop order must be reduce only |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |         | 899              | 1101             | stop3-6   | stop order must be reduce only |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | reduce  | 899              | 1101             | stop4-1   |       |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | post    | 899              | 1101             | stop4-2   | stop order must be reduce only |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |         | 899              | 1101             | stop4-3   | stop order must be reduce only |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | reduce  | 899              | 1101             | stop4-4   |    |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | post    | 899              | 1101             | stop4-5   | stop order must be reduce only |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |         | 899              | 1101             | stop4-6   | stop order must be reduce only |

      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | reduce  | 899              | 1101             | stop5-1   |    |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | post    | 899              | 1101             | stop5-2   | stop order must be reduce only |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |         | 899              | 1101             | stop5-3   | stop order must be reduce only |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | reduce  | 899              | 1101             | stop5-4   |    |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | post    | 899              | 1101             | stop5-5   | stop order must be reduce only |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |         | 899              | 1101             | stop5-6   | stop order must be reduce only |

      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | reduce  | 899              | 1101             | stop6-1   |    |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | post    | 899              | 1101             | stop6-2   | stop order must be reduce only |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN |         | 899              | 1101             | stop6-3   | stop order must be reduce only |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-4   |    |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | post    | 899              | 1101             | stop6-5   | stop order must be reduce only |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         |         | 899              | 1101             | stop6-6   | stop order must be reduce only |


  Scenario:2b Make sure we can send sell side stop orders with all possible TIFs and order types (MARKET/LIMIT)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Stop orders require active orders or a position so create some orders here
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     |
      | party1  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party2  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party4  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party5  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party6  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party7  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party8  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party9  | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party10 | ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | reduce  | 899              | 1101             | stop1-1   |       |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | post    | 899              | 1101             | stop1-2   | stop order must be reduce only |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |         | 899              | 1101             | stop1-3   | stop order must be reduce only |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | reduce  | 899              | 1101             | stop1-4   |    |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA | post    | 899              | 1101             | stop1-5   | stop order must be reduce only |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |         | 899              | 1101             | stop1-6   | stop order must be reduce only |

      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce  | 899              | 1101             | stop2-1   |    |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | post    | 899              | 1101             | stop2-2   | stop order must be reduce only |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |         | 899              | 1101             | stop2-3   | stop order must be reduce only |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce  | 899              | 1101             | stop2-4   |    |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | post    | 899              | 1101             | stop2-5   | stop order must be reduce only |
      | party2| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |         | 899              | 1101             | stop2-6   | stop order must be reduce only |

      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | reduce  | 899              | 1101             | stop3-1   |    |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN | post    | 899              | 1101             | stop3-2   | stop order must be reduce only |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN |         | 899              | 1101             | stop3-3   | stop order must be reduce only |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop3-4   |    |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | post    | 899              | 1101             | stop3-5   | stop order must be reduce only |
      | party3| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |         | 899              | 1101             | stop3-6   | stop order must be reduce only |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | fb price trigger | ra price trigger | reference | error |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | reduce  | 899              | 1101             | stop4-1   |       |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK | post    | 899              | 1101             | stop4-2   | stop order must be reduce only |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |         | 899              | 1101             | stop4-3   | stop order must be reduce only |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | reduce  | 899              | 1101             | stop4-4   |    |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA | post    | 899              | 1101             | stop4-5   | stop order must be reduce only |
      | party4| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |         | 899              | 1101             | stop4-6   | stop order must be reduce only |

      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | reduce  | 899              | 1101             | stop5-1   |    |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC | post    | 899              | 1101             | stop5-2   | stop order must be reduce only |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |         | 899              | 1101             | stop5-3   | stop order must be reduce only |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | reduce  | 899              | 1101             | stop5-4   |    |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC | post    | 899              | 1101             | stop5-5   | stop order must be reduce only |
      | party5| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |         | 899              | 1101             | stop5-6   | stop order must be reduce only |

      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | reduce  | 899              | 1101             | stop6-1   |    |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN | post    | 899              | 1101             | stop6-2   | stop order must be reduce only |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFN |         | 899              | 1101             | stop6-3   | stop order must be reduce only |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-4   |    |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | post    | 899              | 1101             | stop6-5   | stop order must be reduce only |
      | party6| ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         |         | 899              | 1101             | stop6-6   | stop order must be reduce only |





  Scenario:3 Make sure we cannot send buy side stop orders with all possible TIFs and order types (MARKET/LIMIT) while in auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party1  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party2  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party3  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party4  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party5  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party6  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party7  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party8  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party9  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party10 | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party1  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop1-1   | stop orders are not accepted during the opening auction   |
      | party2  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |            | reduce  | 899              | 1101             | stop2-1   | stop orders are not accepted during the opening auction   |
      | party3  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |            | reduce  | 899              | 1101             | stop3-1   | stop orders are not accepted during the opening auction   |
      | party4  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |            | reduce  | 899              | 1101             | stop4-1   | stop orders are not accepted during the opening auction   |
      | party5  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |            | reduce  | 899              | 1101             | stop5-1   | stop orders are not accepted during the opening auction   |
      | party6  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-1   | stop orders are not accepted during the opening auction   |
      | party7  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |            | reduce  | 899              | 1101             | stop7-1   | stop orders are not accepted during the opening auction   |
      | party8  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction   |
      | party9  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction   |
      | party10 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |            | reduce  | 899              | 1101             | stop10-1  | stop orders are not accepted during the opening auction   |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"






  Scenario:3b Make sure we cannot send buy side stop orders with all possible TIFs and order types (MARKET/LIMIT) while in auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party1  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party2  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party3  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party4  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party5  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party6  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party7  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party8  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party9  | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |
      | party10 | ETH/DEC23 | sell | 1     | 900   | 0                 | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party1  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop1-1   | stop orders are not accepted during the opening auction |
      | party2  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |            | reduce  | 899              | 1101             | stop2-1   | stop orders are not accepted during the opening auction |
      | party3  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |            | reduce  | 899              | 1101             | stop3-1   | stop orders are not accepted during the opening auction |
      | party4  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |            | reduce  | 899              | 1101             | stop4-1   | stop orders are not accepted during the opening auction |
      | party5  | ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |            | reduce  | 899              | 1101             | stop5-1   | stop orders are not accepted during the opening auction |
      | party6  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-1   | stop orders are not accepted during the opening auction |
      | party7  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |            | reduce  | 899              | 1101             | stop7-1   | stop orders are not accepted during the opening auction |
      | party8  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction |
      | party9  | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction |
      | party10 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |            | reduce  | 899              | 1101             | stop10-1  | stop orders are not accepted during the opening auction |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"



  Scenario:4 Make sure we cannot send sell side stop orders with all possible TIFs and order types (MARKET/LIMIT) while in auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party1  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party3  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party6  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party7  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party8  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party9  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party10 | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party1  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop1-1   | stop orders are not accepted during the opening auction   |
      | party2  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |            | reduce  | 899              | 1101             | stop2-1   | stop orders are not accepted during the opening auction   |
      | party3  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |            | reduce  | 899              | 1101             | stop3-1   | stop orders are not accepted during the opening auction   |
      | party4  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |            | reduce  | 899              | 1101             | stop4-1   | stop orders are not accepted during the opening auction   |
      | party5  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |            | reduce  | 899              | 1101             | stop5-1   | stop orders are not accepted during the opening auction   |
      | party6  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-1   | stop orders are not accepted during the opening auction   |
      | party7  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |            | reduce  | 899              | 1101             | stop7-1   | stop orders are not accepted during the opening auction   |
      | party8  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction   |
      | party9  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction   |
      | party10 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |            | reduce  | 899              | 1101             | stop10-1  | stop orders are not accepted during the opening auction   |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"


  Scenario:4b Make sure we cannot send sell side stop orders with all possible TIFs and order types (MARKET/LIMIT) while in auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount   |
      | party1  | ETH   | 10000000 |
      | party2  | ETH   | 10000000 |
      | party3  | ETH   | 10000000 |
      | party4  | ETH   | 10000000 |
      | party5  | ETH   | 10000000 |
      | party6  | ETH   | 10000000 |
      | party7  | ETH   | 10000000 |
      | party8  | ETH   | 10000000 |
      | party9  | ETH   | 10000000 |
      | party10 | ETH   | 10000000 |
      | lp1     | ETH   | 10000000 |
      | lphelp1 | ETH   | 10000000 |
      | lphelp2 | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | party1  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party3  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party6  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party7  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party8  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party9  | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party10 | ETH/DEC23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type        | tif     | expires in | only    | fb price trigger | ra price trigger | reference | error |
      | party1  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce  | 899              | 1101             | stop1-1   | stop orders are not accepted during the opening auction   |
      | party2  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |            | reduce  | 899              | 1101             | stop2-1   | stop orders are not accepted during the opening auction   |
      | party3  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |            | reduce  | 899              | 1101             | stop3-1   | stop orders are not accepted during the opening auction   |
      | party4  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |            | reduce  | 899              | 1101             | stop4-1   | stop orders are not accepted during the opening auction   |
      | party5  | ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |            | reduce  | 899              | 1101             | stop5-1   | stop orders are not accepted during the opening auction   |
      | party6  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTT | 50         | reduce  | 899              | 1101             | stop6-1   | stop orders are not accepted during the opening auction   |
      | party7  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GTC |            | reduce  | 899              | 1101             | stop7-1   | stop orders are not accepted during the opening auction   |
      | party8  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_IOC |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction   |
      | party9  | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_FOK |            | reduce  | 899              | 1101             | stop8-1   | stop orders are not accepted during the opening auction   |
      | party10 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT  | TIF_GFA |            | reduce  | 899              | 1101             | stop10-1  | stop orders are not accepted during the opening auction   |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | lphelp1 | ETH/DEC23 | buy  | 110    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 110    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp1 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lphelp2 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"
