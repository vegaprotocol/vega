
Feature: Set up a spot market, with an opening auction, then uncross the book. Make sure opening auction can end.
    Background:
        Given the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.005     | 0.002              |
        And the simple risk model named "my-simple-risk-model":
            | long                   | short                  | max move up | min move down | probability of trading |
            | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
        And the fees configuration named "my-fees-config":
            | maker fee | infrastructure fee |
            | 0.004     | 0.001              |
        And the spot markets:
            | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    |
            | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 5                | fees-config-1 | default-none     | default-basic |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount     |
            | party1 | ETH   | 1000000000 |
            | party2 | ETH   | 1000000000 |
            | party2 | BTC   | 5          |
            | party4 | ETH   | 10000000   |
            | party5 | BTC   | 10         |

    Scenario: Party submits pegged orders and they are getting priced and repriced as needed
        Given the following network parameters are set:
            | name                           | value |
            | limits.markets.maxPeggedOrders | 10    |

        # place orders and generate trades
        When the parties place the following orders:
            | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
            | party2 | BTC/ETH   | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
            | party1 | BTC/ETH   | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
            | party2 | BTC/ETH   | sell | 5      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

        And the parties place the following pegged orders:
            | party  | market id | side | volume | pegged reference | offset |
            | party4 | BTC/ETH   | buy  | 1      | BID              | 100    |
            | party4 | BTC/ETH   | buy  | 1      | MID              | 100    |
            | party5 | BTC/ETH   | sell | 1      | ASK              | 100    |
            | party5 | BTC/ETH   | sell | 1      | MID              | 100    |

        Then the pegged orders should have the following states:
            | party  | market id | side | volume | reference | offset | price | status        |
            | party4 | BTC/ETH   | buy  | 1      | BID       | 100    | 0     | STATUS_PARKED |
            | party4 | BTC/ETH   | buy  | 1      | MID       | 100    | 0     | STATUS_PARKED |
            | party5 | BTC/ETH   | sell | 1      | ASK       | 100    | 0     | STATUS_PARKED |
            | party5 | BTC/ETH   | sell | 1      | MID       | 100    | 0     | STATUS_PARKED |

        When the opening auction period ends for market "BTC/ETH"
        Then the market data for the market "BTC/ETH" should be:
            | mark price | trading mode            | auction trigger             |
            | 1000000    | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

        And the pegged orders should have the following states:
            | party  | market id | side | volume | reference | offset | price   | status        |
            | party4 | BTC/ETH   | buy  | 1      | BID       | 100    | 949900  | STATUS_ACTIVE |
            | party4 | BTC/ETH   | buy  | 1      | MID       | 100    | 974900  | STATUS_ACTIVE |
            | party5 | BTC/ETH   | sell | 1      | ASK       | 100    | 1000100 | STATUS_ACTIVE |
            | party5 | BTC/ETH   | sell | 1      | MID       | 100    | 975100  | STATUS_ACTIVE |

        And the order book should have the following volumes for market "BTC/ETH":
            | side | price   | volume |
            | sell | 975100  | 1      |
            | sell | 1000000 | 4      |
            | sell | 1000100 | 1      |
            | buy  | 974900  | 1      |
            | buy  | 950000  | 1      |
            | buy  | 949900  | 1      |


        # Move the best bid and assure the orders are repriced
        When the parties amend the following orders:
            | party  | reference | price  | size delta | tif     |
            | party2 | t2-b-1    | 970000 | 0          | TIF_GTC |

        Then the pegged orders should have the following states:
            | party  | market id | side | volume | reference | offset | price   | status        |
            | party4 | BTC/ETH   | buy  | 1      | BID       | 100    | 969900  | STATUS_ACTIVE |
            | party4 | BTC/ETH   | buy  | 1      | MID       | 100    | 984900  | STATUS_ACTIVE |
            | party5 | BTC/ETH   | sell | 1      | ASK       | 100    | 1000100 | STATUS_ACTIVE |
            | party5 | BTC/ETH   | sell | 1      | MID       | 100    | 985100  | STATUS_ACTIVE |

        And the order book should have the following volumes for market "BTC/ETH":
            | side | price   | volume |
            | sell | 1000100 | 1      |
            | sell | 985100  | 1      |
            | sell | 1000000 | 4      |
            | buy  | 970000  | 1      |
            | buy  | 969900  | 1      |
            | buy  | 984900  | 1      |

        # example for submitting LP for spots
        When the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee | lp type    |
            | lp1 | party1 | BTC/ETH   | 1000              | 0.1 | submission |
