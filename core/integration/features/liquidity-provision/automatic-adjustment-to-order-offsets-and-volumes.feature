
Feature: Confirm automatic adjustments to LP orders as specified in 0038-OLIQ

    Background:
      Given the log normal risk model named "lognormal-risk-model-fish":
        | risk aversion | tau  | mu | r     | sigma |
        | 0.001         | 0.01 | 0  | 0.0   | 2     |
      And the price monitoring named "price-monitoring-1":
        | horizon | probability       | auction extension |
        | 1000000 | 0.999999999999999 | 300               |
      And the margin calculator named "margin-calculator-1":
        | search factor | initial factor | release factor |
        | 1.2           | 1.5            | 2              |

      And the markets:
        | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring   | data source config     |
        | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | price-monitoring-1 | default-eth-for-future |

      And the following network parameters are set:
        | name                                                   | value |
        | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.01  |
        | network.markPriceUpdateMaximumFrequency                | 0s    |
      And the average block duration is "1"

    Scenario: When offsets result in prices outside of price monitoring bounds orders get automatically placed at the bounds (0038-OLIQ-009), modifying market.liquidity.minimum.probabilityOfTrading.lpOrders adjusts the volumes after reference move 0038-OLIQ-007)

      Given the parties deposit on asset's general account the following amount:
        | party            | asset | amount                 |
        | aux              | USD   |          1000000000000 |
        | aux2             | USD   |          1000000000000 |
        | lp               | USD   | 1000000000000000000000 |

      When the parties submit the following liquidity provision:
        | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
        | lp1 | lp     | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 139    | submission |
        | lp1 | lp     | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 3000   | submission |
    
      Then the parties place the following orders:
        | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
        | aux   | ETH/DEC19 | buy  | 10     | 140   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
        | aux2  | ETH/DEC19 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
        | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
        | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      Then the opening auction period ends for market "ETH/DEC19"
      And the market data for the market "ETH/DEC19" should be:
        | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
        | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1000000 | 9         | 2465      |

      And the liquidity provisions should have the following states:
        | id  | party | market    | commitment amount | status        |
        | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |

      # Observe that given specified pegs we should have an LP buy order placed at a price of 1 and sell order placed at a price of 3160, however, since both of these fall outside of price monitoring bounds the orders gets moved accordingly (0038-OLIQ-009)
      Then the order book should have the following volumes for market "ETH/DEC19":
        | side | price | volume  |
        | sell | 2465  |    3652 |
        | sell | 160   |      10 |
        | buy  | 140   |      10 |
        | buy  | 9     | 1000000 |

      When the following network parameters are set:
        | name                                                   | value |
        | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.1   |
      And the network moves ahead "10" blocks

      # updating the parameter itself is not enough for the volumes to get affected
      Then the order book should have the following volumes for market "ETH/DEC19":
        | side | price | volume  |
        | sell | 2465  |    3652 |
        | sell | 160   |      10 |
        | buy  | 140   |      10 |
        | buy  | 9     | 1000000 |

      # update the LP peg reference to trigger an update
      When the parties amend the following orders:
        | party | reference | price | size delta | tif     |
        | aux   | bestBid   | 141   | 0          | TIF_GTC |

      # observe volumes drop
      Then the order book should have the following volumes for market "ETH/DEC19":
        | side | price | volume |
        | sell | 2465  |    366 |
        | sell | 160   |     10 |
        | buy  | 141   |     10 |
        | buy  | 9     | 100000 |

      When the parties amend the following orders:
        | party | reference  | price | size delta | tif     |
        | aux2   | bestOffer | 155   | 0          | TIF_GTC |

      # volumes unaffected by further peg change - they are already getting pushed in by price monitoring bounds and network parameter hasn't been update
      Then the order book should have the following volumes for market "ETH/DEC19":
        | side | price | volume |
        | sell |  2465 |    366 |
        | sell |   155 |     10 |
        | buy  |   141 |     10 |
        | buy  |     9 | 100000 |

      # now decrease the minimum probability of trading
      When the following network parameters are set:
        | name                                                   | value |
        | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.05  |
      And the network moves ahead "10" blocks
      # again, no immediate change in volumes as reference prices didn't move and no trades occurred
      Then the order book should have the following volumes for market "ETH/DEC19":
        | side | price | volume |
        | sell |  2465 |    366 |
        | sell |   155 |     10 |
        | buy  |   141 |     10 |
        | buy  |     9 | 100000 |

      Then the parties place the following orders:
        | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
        | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
        | aux2  | ETH/DEC19 | sell | 1      | 150   | 1                | TYPE_LIMIT | TIF_GTC |           |

      Then the order book should have the following volumes for market "ETH/DEC19":
        | side | price | volume |
        | sell |  2465 |    731 |
        | sell |   155 |     10 |
        | buy  |   141 |     10 |
        | buy  |     9 | 200000 |