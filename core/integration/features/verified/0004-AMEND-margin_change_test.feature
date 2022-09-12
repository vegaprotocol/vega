Feature: Amend orders

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | USD   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: 001 Amend rejected for non existing order
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | USD   | 10000  |
      | aux    | USD   | 100000 |
      | aux2   | USD   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1  | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC | party1-ref-1 |

# cancel the order, so we cannot edit it.
    And the parties cancel the following orders:
      | party | reference   |
      | party1  | party1-ref-1 |

    Then the parties amend the following orders:
      | party | reference   | price | size delta | tif     | error                        |
      | party1  | party1-ref-1 | 2     | 3          | TIF_GTC | OrderError: Invalid Order ID |

  Scenario: 002 Reduce size success and not loosing position in order book
# setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | USD   | 1000000  |
      | party2 | USD   | 1000000  |
      | party3 | USD   | 1000000  |
      | aux    | USD   | 1000000  |
      | aux2   | USD   | 1000000  |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux2   | ETH/DEC19 | 50000             | 0.001 | sell | ASK              | 500        | 1      | submission |
      | lp1 | aux2   | ETH/DEC19 | 50000             | 0.001 | buy  | BID              | 500        | 1      | amendment  |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 50001| 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | buy  | 1      | 2000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 2000 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
  
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 2000       | TRADING_MODE_CONTINUOUS | 0            | 50000          | 1             |
    # party 123 plalces orders on the book
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC19 | sell | 5      | 2100  | 0                | TYPE_LIMIT | TIF_GTC | party1-ref-1 |
      | party2 | ETH/DEC19 | sell | 7      | 2200  | 0                | TYPE_LIMIT | TIF_GTC | party2-ref-2 |
      | party3 | ETH/DEC19 | buy  | 4      | 1900  | 0                | TYPE_LIMIT | TIF_GTC | party3-ref-3 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party1 | USD   | ETH/DEC19 | 0      | 1000000    | 0     |
      | party2 | USD   | ETH/DEC19 | 0      | 1000000    | 0     |
      | party3 | USD   | ETH/DEC19 | 0      | 1000000    | 0     |
 
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party3 | ETH/DEC19 | 0           | 0      | 0       | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    # trigger a new mark price 
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party3 | ETH/DEC19 | buy  | 2      | 2100  | 1                | TYPE_LIMIT | TIF_GTC | party3-ref-1 |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 2100       | TRADING_MODE_CONTINUOUS | 0            | 50000          | 3             |

    Then the following trades should be executed:
      | buyer  | price  | size | seller |
      | party3 | 2100   | 2    | party1 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party1 | USD   | ETH/DEC19 | 0      | 1000000  | 0     |
      | party2 | USD   | ETH/DEC19 | 0      | 1000000  | 0     |
      | party3 | USD   | ETH/DEC19 | 1600   | 998395   | 0     |
 
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party3 | ETH/DEC19 | 400         | 1280   | 1600    | 2000    |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | 0              | 0            |
      | party2 | 0      | 0              | 0            |
      | party3 | 2      | 0              | 0            |

    # reducing size
    Then the parties amend the following orders:
      | party | reference    | price | size delta | tif     |
      | party3| party3-ref-3 | 0     | -1         | TIF_GTC |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party1 | USD   | ETH/DEC19 | 0      | 1000000  | 0     |
      | party2 | USD   | ETH/DEC19 | 0      | 1000000  | 0     |
      | party3 | USD   | ETH/DEC19 | 1600   | 998395   | 0     |
 
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party3 | ETH/DEC19 | 400         | 1280   | 1600    | 2000    |
   
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | 0              | 0            |
      | party2 | 0      | 0              | 0            |
      | party3 | 2      | 0              | 0            |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    # trigger a new trade
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party3 | ETH/DEC19 | buy  | 6      | 2200  | 2                | TYPE_LIMIT | TIF_GTC | party3-ref-1 |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 2101       | TRADING_MODE_CONTINUOUS | 0            | 50000          | 8             |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party1 | USD   | ETH/DEC19 | 0      | 999995   | 0     |
      | party2 | USD   | ETH/DEC19 | 0      | 1000000  | 0     |
      | party3 | USD   | ETH/DEC19 | 6464   | 993522   | 0     |
 
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party2 | ETH/DEC19 | 0           | 0      | 0       | 0       |
      | party3 | ETH/DEC19 | 1616        | 5171   | 6464    | 8080    |
  
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -5     | -5             | 0            |
      | party2 | 0      | 0              | 0            |
      | party3 | 8      | 5              | 0            |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"




