Feature: Simple Spot Order Market fee and infrastructure fee calculation
  Scenario:  Simple Spot Order Market fee and infrastructure fee calculation
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.1       | 0.2                |

    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | 100         | -100          | 1                      |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | price-monitoring | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000000 |
      | party2 | BTC   | 100000   |

    #When the parties submit the following liquidity provision:
    #  | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
    #  | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
    #  | lp1 | lpprov | BTC/ETH   | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party2 | BTC/ETH   | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party1" should have holding account balance of "1000" for asset "ETH"

    Then the orders should have the following states:
      | party  | market id | side | volume | price  | status        |
      | party1 | BTC/ETH   | buy  | 1      | 100000 | STATUS_ACTIVE |
      | party2 | BTC/ETH   | sell | 1      | 100000 | STATUS_ACTIVE |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100000" for the market "BTC/ETH"

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | target stake | supplied stake |
      | 100000     | TRADING_MODE_CONTINUOUS | 0            | 0              |

    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party1 | 100000 | 1    | party2 |

    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party1" should have holding account balance of "0" for asset "ETH"

    Then the parties should have the following account balances:
      | party  | asset | market id | general |
      | party1 | ETH   | BTC/ETH   | 9999000 |
      | party1 | BTC   | BTC/ETH   | 0       |
      | party2 | ETH   | BTC/ETH   | 1000    |
      | party2 | BTC   | BTC/ETH   | 100000  |

#Then the parties should have the following account balances:
#      | party   | asset | market id | margin | general |
#      | party1  | ETH   | BTC/ETH | 1330   | 0    |
#      | party2  | ETH   | BTC/ETH | 718    | 0    |

# @vanitha the exepctation here is wrong. Trades done at the end of opening auctions don't pay fees.
# And the accumulated infrastructure fees should be "500" for the asset "ETH"
# And the accumulated infrastructure fees should be "100" for the asset "BTC"
# And the accumulated liquidity fees should be "20" for the market "BTC/ETH"
