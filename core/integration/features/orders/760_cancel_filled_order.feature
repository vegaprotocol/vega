Feature: Close a filled order twice

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e0                    | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |

  Scenario: Traders place an order, a trade happens, and orders are cancelled after being filled
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | aux              | BTC   | 100000    |
      | aux2             | BTC   | 100000    |
      | aux3             | BTC   | 100000    |
      | aux4             | BTC   | 100000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    # AC 0068-MATC-055
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference | only |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-0     | post |
      | aux   | ETH/DEC19 | buy  | 2      | 2     | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |      |
      | aux2  | ETH/DEC19 | buy  | 4      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |      |
      | aux   | ETH/DEC19 | sell | 4      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |      |
      | aux   | ETH/DEC19 | sell | 4      | 121   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     | post |
      | aux   | ETH/DEC19 | sell | 5      | 122   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     | post |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 1     | 1      |
      | buy  | 2     | 2      |
      | sell | 121   | 4      |
      | sell | 122   | 5      |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux2   | 4      | 0              | 0            |     

    # setup orderbook
    And the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 10     | 120   | 1                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    When the parties cancel the following orders:
      | party           | reference      | error                                  |
      | buySideProvider | buy-provider-1 | unable to find the order in the market |
    When the parties cancel the following orders:
      | party            | reference       | error                                  |
      | sellSideProvider | sell-provider-1 | unable to find the order in the market |
    Then the insurance pool balance should be "0" for the market "ETH/DEC19"

    # AC 0068-MATC-040, 0068-MATC-041, 0068-MATC-042
    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type       | tif     | reference  | only   | expires in | error |
      | aux  | ETH/DEC19 | sell | 2      | 123   | 0                | TYPE_LIMIT | TIF_GTT | postonly-1 | post   | 3600       |       |
      | aux  | ETH/DEC19 | sell | 2      | 124   | 0                | TYPE_LIMIT | TIF_GTC | postonly-2 | post   |            |       |
      | aux3 | ETH/DEC19 | buy  | 2      | 123   | 0                | TYPE_LIMIT | TIF_GTT | postonly-3 | post   | 3600       | OrderError: post only order would trade |
      | aux3 | ETH/DEC19 | buy  | 1      | 123   | 0                | TYPE_LIMIT | TIF_GTT | postonly-4 | post   | 3600       | OrderError: post only order would trade |
      | aux4 | ETH/DEC19 | buy  | 2      | 124   | 0                | TYPE_LIMIT | TIF_GTC | postonly-5 | post   |            | OrderError: post only order would trade |
      | aux4 | ETH/DEC19 | buy  | 1      | 124   | 0                | TYPE_LIMIT | TIF_GTC | postonly-6 | post   |            | OrderError: post only order would trade |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 1     | 1      |
      | buy  | 2     | 2      |
      | buy  | 123   | 0      |
      | buy  | 124   | 0      |
      | sell | 121   | 4      |
      | sell | 122   | 5      |
      | sell | 123   | 2      |
      | sell | 124   | 2      |

    # AC 0068-MATC-056:Incoming [MARKET] reduce-only orders which reduce the trader's absolute position will be matched against the opposite side of the book
    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type        | tif     | reference   | only    | error |
      | aux2 | ETH/DEC19 | sell | 1      | 130   | 1                | TYPE_MARKET | TIF_IOC | reduceonly-1| reduce  |       |
      
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux2   | 3      | -354           | -118         |    

    # AC 0068-MATC-043
    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type        | tif     | reference   | only    | error |
      | aux2 | ETH/DEC19 | sell | 4      | 130   | 2                | TYPE_MARKET | TIF_IOC | reduceonly-1| reduce  |       |
      
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | -1     | 119            | 355          | 
      | aux2   | 1      | -119           | -355         |    
       
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 130   | 0      |   

    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type        | tif     | reference | only   | error |
      | aux3 | ETH/DEC19 | buy  | 40     | 2    | 0                | TYPE_LIMIT | TIF_GTC   | ref-6     |        |       |
      | aux3 | ETH/DEC19 | sell | 40     | 120  | 0                | TYPE_LIMIT | TIF_GTC   | ref-7     |        |       |

    And then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # AC 0068-MATC-044; 0068-MATC-045; 0068-MATC-046; 0068-MATC-047
    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type        | tif     | reference   | only   | error |
      | aux2 | ETH/DEC19 | buy  | 4      | 1     | 0                | TYPE_MARKET | TIF_IOC | reduceonly-2| reduce | OrderError: reduce only order would not reduce position  | 
      | aux2 | ETH/DEC19 | buy  | 4      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | reduceonly-3| reduce | OrderError: reduce only order would not reduce position  | 
      | aux2 | ETH/DEC19 | sell | 1      | 130   | 0                | TYPE_LIMIT  | TIF_GTC | reduceonly-4| reduce |       | 
      | aux  | ETH/DEC19 | buy  | 1      | 130   | 1                | TYPE_MARKET | TIF_FOK | reduceonly-5| reduce |       | 
   
    And then the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | 0      | 0              | 355          | 
      | aux2   | 1      | 0              | -355         |    

    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type        | tif     | reference | only   | error |
      | aux  | ETH/DEC19 | buy  | 100    | 120   | 1                | TYPE_LIMIT  | TIF_GTC | ref-8     |        |       |
      | aux2 | ETH/DEC19 | sell | 10     | 120   | 1                | TYPE_LIMIT  | TIF_GTC | ref-9     |        |       |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | 49     | 0              | 355          | 
      | aux2   | -9     | 0              | -355         |  

    # AC 0068-MATC-048; 0068-MATC-049; 0068-MATC-050; 0068-MATC-051
    When the parties place the following orders with ticks:
      | party| market id | side | volume | price | resulting trades | type        | tif     | reference   | only   | error |
      | aux2 | ETH/DEC19 | sell | 1      | 22    | 0                | TYPE_LIMIT  | TIF_FOK | reduceonly-6| reduce | OrderError: reduce only order would not reduce position | 
      | aux2 | ETH/DEC19 | buy  | 1      | 120   | 0                | TYPE_LIMIT  | TIF_FOK | reduceonly-7| reduce |       | 
      | aux2 | ETH/DEC19 | sell | 1      | 22    | 0                | TYPE_MARKET | TIF_FOK | reduceonly-8| reduce | OrderError: reduce only order would not reduce position | 
      | aux2 | ETH/DEC19 | buy  | 9      | 22    | 2                | TYPE_MARKET | TIF_FOK | reduceonly-9| reduce |       | 

      Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | 40     | 80             | 369          | 
      | aux2   | 0      | 0              | -369         |  