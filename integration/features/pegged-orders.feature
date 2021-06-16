Feature: Test pegged orders

  Background:

    Given the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Test the basics  
    Given the following network parameters are set: 
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 1     |
    And the average block duration is "1"

    Given the price monitoring updated every "60" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 60                |
    And the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.11 | 0.1   | 10          | -11           | 0.1                    |
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC21 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future |
    And the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | trader1   | ETH   | 100000000    |
      | trader2   | ETH   | 100000000    |
      | trader3   | ETH   | 100000000    |
      | aux       | ETH   | 100000000000 |

    Given the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | aux   | ETH/DEC21 | 110               | 0.001 | buy        | BID             | 500              | -10          |
      | lp1 | aux   | ETH/DEC21 | 110               | 0.001 | sell       | ASK             | 500              | 10           |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux     | ETH/DEC21 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | aux-buy   |
      | aux     | ETH/DEC21 | sell | 1      | 190   | 0                | TYPE_LIMIT | TIF_GTC | aux-sell  |
      
    # These pegged orders get placed and parked since market is in auction
    Then the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset |
      | trader1 | ETH/DEC21 | sell | 10     | MID       |     13 |
      | trader2 | ETH/DEC21 | buy  | 5      | BID       |      0 |

    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 0     | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 0     | STATUS_PARKED |

    # Trigger an auction end set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-1 |
      | trader2 | ETH/DEC21 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-1 |
    Then the opening auction period ends for market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "100" for the market "ETH/DEC21"
 
    # TODO: Add handling of TIF (and order type?) and test cases
    When the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset  |
      | trader2 | ETH/DEC21 | sell | 10     | MID       |     12  |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       |    -15  |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       |     -2  |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       |      3  |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       |      0  |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       |      0  |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       |     -9  |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       |     -10 |
  
    # Check pegged orders behave as expected (can have 0 offset with bid/ask, orders with resulting price that's invalid get parked)
    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 113     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 10      | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 12      | 112     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       | -15     | 85      | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       | -2      | 8       | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       | 3       | 193     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       | 0       | 10      | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       | 0       | 190     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 | 1000190 | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       | -9      | 1       | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       | -10     | 0       | STATUS_PARKED |
    
    #Check that MID reference cannot be combined with a 0 offset. 
    And the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset  | error                                        |
      | trader1 | ETH/DEC21 | sell | 1      | MID       |      0  | OrderError: offset must be greater than zero |

    #Check that BID reference cannot be combined with a positive offset. 
    And the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset  | error                           |
      | trader2 | ETH/DEC21 | buy  | 1      | BID       |      1  | OrderError: offset must be <= 0 |

    #Check that ASK reference cannot be combined with a negative offset. 
    And the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset  | error                           |
      | trader1 | ETH/DEC21 | sell | 1      | ASK       |     -1  | OrderError: offset must be >= 0 |
      
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100        | TRADING_MODE_CONTINUOUS | 60      | 89        | 110       | 110          | 110            | 1             |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 2      | 111   | 0                | TYPE_LIMIT | TIF_GTC | trader1-2 |

    #TODO: If MID is fractional, eg. 150.5 and MID ref is used it gets rounded UP for buy and DOWN for sell - verify that it's captured in the spec
    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 163     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 111     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 12      | 162     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       | -15     | 136     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       | -2      | 109     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       | 3       | 193     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       | 0       | 111     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       | 0       | 190     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 | 1000190 | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       | -9      | 102     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       | -10     | 101     | STATUS_ACTIVE |

    # Go into price auction and check that orders get parked
    # TODO: resutling trades check doesn't seem to be working, expecting 0 trades, but any value (left at incorrect 123 now) results in a pass
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC21 | sell | 2      | 111   | 123              | TYPE_LIMIT | TIF_GTC | trader2-2 |

    # TODO: output formatting is sometimes wrong (expected and actual value the wrong way around)
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
      | 100        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 110          | 110            | 1             |

    #TODO: I think we need something like this to make sure that all orders have been specified in the next step
    Then there should be 9 pegged orders

    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 0       | STATUS_PARKED |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 12      | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       | -15     | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       | -2      | 0       | STATUS_PARKED |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       | 3       | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       | 0       | 0       | STATUS_PARKED |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       | 0       | 0       | STATUS_PARKED |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       | -9      | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       | -10     | 0       | STATUS_PARKED |

    #Extend with liquidity auction and check that orders get unparked
    When the network moves ahead "61" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | extension trigger         | target stake | supplied stake | open interest |
      | 100        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_LIQUIDITY | 110          | 110            | 1             |
   
    # TODO (WG): Am I missing something here? If at the end of price auction we don't have sufficient liquidity to uncross we should never leave auction, just extend it with liquidity
    # The auction isn't extended, we only check liquidity at the end of the price auction, at which point
    # There's no reason to extend
    And the auction extension trigger should be "AUCTION_TRIGGER_LIQUIDITY" for market "ETH/DEC21"

    Then the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset |
      | trader1 | ETH/DEC21 | sell | 17     | MID       |     14 |

    And the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader1 | ETH/DEC21 | sell | 17     | MID       |     14  | 0       | STATUS_PARKED |

    Then the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | aux   | ETH/DEC21 | 500               | 0.001 | buy        | BID             | 500              | -10          |
      | lp1 | aux   | ETH/DEC21 | 500               | 0.001 | sell       | ASK             | 500              | 10           |
    
    When the network moves ahead "1" blocks
    
    #TODO: Perhaps we need a "clear trades" statement or somehow autoclear it when time advances? I can see the trade in "debug trades" output, but there's also the previous 1@100 one there
    #Then debug trades
    Then the auction ends with a traded volume of "2" at a price of "111"

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest | best static bid price | best static ask price |
      | 111        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 367          | 500            | 3             | 10                    | 190                   |

    And the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 113     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 10      | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 12      | 112     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       | -15     | 85      | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       | -2      | 8       | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       | 3       | 193     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       | 0       | 10      | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       | 0       | 190     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 | 1000190 | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       | -9      | 1       | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       | -10     | 0       | STATUS_PARKED |

    # Check orders get parked as their price becomes invalid 
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 1      | 5     | 0                | TYPE_LIMIT | TIF_GTC | trader1-4 |
      | trader2 | ETH/DEC21 | sell | 1      | 185   | 0                | TYPE_LIMIT | TIF_GTC | trader2-4 |

    And the market data for the market "ETH/DEC21" should be:
      | best static bid price | best static ask price | static mid price | best static bid volume | best static ask volume |
      | 10                    | 185                   | 97               | 1                      | 1                      |

    And the traders cancel the following orders:
      | trader  | reference |
      | aux     | aux-buy   |

    And the market data for the market "ETH/DEC21" should be:
      | best static bid price | best static ask price | static mid price | best static bid volume | best static ask volume |
      | 5                     | 185                   | 95               | 1                      | 1                      |

    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 108     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 5       | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 12      | 107     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       | -15     | 80      | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       | -2      | 3       | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       | 3       | 188     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       | 0       | 5       | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       | 0       | 185     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 | 1000185 | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       | -9      | 0       | STATUS_PARKED |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       | -10     | 0       | STATUS_PARKED |

    #TODO: Both of these groups cannot result in passes, expecting the prices defined above, the ones below are old and hence should now result in a fail
    Then the pegged orders should have the following states:
      | trader  | market id | side | volume | reference | offset  | price   | status        |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 13      | 113     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | BID       | 0       | 10      | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 10     | MID       | 12      | 112     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 5      | MID       | -15     | 85      | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 3      | BID       | -2      | 8       | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 2      | ASK       | 3       | 193     | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 8      | BID       | 0       | 10      | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 11     | ASK       | 0       | 190     | STATUS_ACTIVE |
      | trader2 | ETH/DEC21 | sell | 19     | ASK       | 1000000 | 1000190 | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 1      | BID       | -9      | 1       | STATUS_ACTIVE |
      | trader1 | ETH/DEC21 | buy  | 16     | BID       | -10     | 0       | STATUS_PARKED |

    # TODO: mid price comes back as 93
    And the market data for the market "ETH/DEC21" should be:
      | best bid price | best ask price | mid price | best bid volume | best ask volume |
      | 80             | 185            | 193        | 5               | 11              |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC21 | sell | 2      | 111   | 123              | TYPE_LIMIT | TIF_FOK | trader2-5 |
    

