
Feature: Confirm automatic adjustments to LP orders as specified in 0038-OLIQ

  Background:
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 2     |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability       | auction extension |
      | 1000000 | 0.999999999999999 | 300               |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | price-monitoring-1 | default-eth-for-future | 0.5            | 1e6                    | 1e6                       |

    And the following network parameters are set:
      | name                                                   | value |
      | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.01  |
      | network.markPriceUpdateMaximumFrequency                | 0s    |
      | limits.markets.maxPeggedOrders                         | 10    |
    And the average block duration is "1"

  Scenario: When offsets result in prices outside of price monitoring bounds orders get automatically placed at the bounds, modifying market.liquidity.minimum.probabilityOfTrading.lpOrders adjusts the volumes after reference move) + assure that regular (non-LP) pegged orders don't get affected by the LP price range

    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 139    | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 3000   | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 140   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux2  | ETH/DEC19 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2  | ETH/DEC19 | buy  | 3      | BID              | 139    |
      | aux2  | ETH/DEC19 | sell | 2      | ASK              | 3000   |

    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | best static bid price | static mid price | best static offer price |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1000000 | 9         | 2465      | 140                   | 150              | 160                     |
    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |


    # Observe that given specified pegs we should have an LP buy order placed at a price of 1 and sell order placed at a price of 3160, however, since both of these fall outside of LP price range the orders gets moved accordingly
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 400    |
      | sell | 160   | 10     |
      | buy  | 140   | 10     |
      | buy  | 75    | 1200   |

    # Observe that regular (non-LP) pegged orders unaffected by the LP price range
    Then the pegged orders should have the following states:
      | party | market id | side | volume | reference | offset | price | status        |
      | aux2  | ETH/DEC19 | buy  | 3      | BID       | 139    | 1     | STATUS_ACTIVE |
      | aux2  | ETH/DEC19 | sell | 2      | ASK       | 3000   | 3160  | STATUS_ACTIVE |

    When the following network parameters are set:
      | name                                                   | value |
      | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.1   |
    And the network moves ahead "10" blocks
    # this parameter is no longer meant to have any effect on order sizes
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 400    |
      | sell | 160   | 10     |
      | buy  | 140   | 10     |
      | buy  | 75    | 1200   |

    # update the LP peg reference to trigger an update
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux   | bestBid   | 141   | 0          | TIF_GTC |
    Then the market data for the market "ETH/DEC19" should be:
      | best static bid price | static mid price | best static offer price |
      | 141                   | 150              | 160                     |
    # LP price range is now [0.5*150.5,1.5*150.5]=[75.2,225.7]= (after ceil/floor) [76,225]
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 400    |
      | sell | 160   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 1185   |
      | buy  | 75    | 0      |

    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux2  | bestOffer | 155   | 0          | TIF_GTC |
    Then the market data for the market "ETH/DEC19" should be:
      | best static bid price | static mid price | best static offer price |
      | 141                   | 148              | 155                     |
    # LP price range is now [0.5*148,1.5*148]=[74,222]
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 0      |
      | sell | 222   | 406    |
      | sell | 160   | 0      |
      | sell | 155   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 0      |
      | buy  | 75    | 0      |
      | buy  | 74    | 1217   |

    # now decrease the minimum probability of trading
    When the following network parameters are set:
      | name                                                   | value |
      | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.05  |
    And the network moves ahead "10" blocks
    # again, this parameter is no longer meant to have any effect on order sizes
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 0      |
      | sell | 222   | 406    |
      | sell | 160   | 0      |
      | sell | 155   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 0      |
      | buy  | 75    | 0      |
      | buy  | 74    | 1217   |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 1                | TYPE_LIMIT | TIF_GTC |           |

    # trade doesn't change any of the pegs so orderbook composition is unaffected
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 0      |
      | sell | 222   | 406    |
      | sell | 160   | 0      |
      | sell | 155   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 0      |
      | buy  | 75    | 0      |
      | buy  | 74    | 1217   |

  Scenario:
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 139    | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 3000   | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 140   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux2  | ETH/DEC19 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | best static bid price | static mid price | best static offer price |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1000000 | 9         | 2465      | 140                   | 150              | 160                     |
    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |


    # Observe that given specified pegs we should have an LP buy order placed at a price of 1 and sell order placed at a price of 3160, however, since both of these fall outside of LP price range the orders gets moved accordingly
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 400    |
      | sell | 160   | 10     |
      | buy  | 140   | 10     |
      | buy  | 75    | 1200   |

    When the following network parameters are set:
      | name                                                   | value |
      | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.1   |
    And the network moves ahead "10" blocks
    # this parameter is no longer meant to have any effect on order sizes
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 400    |
      | sell | 160   | 10     |
      | buy  | 140   | 10     |
      | buy  | 75    | 1200   |

    # update the LP peg reference to trigger an update
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux   | bestBid   | 141   | 0          | TIF_GTC |
    Then the market data for the market "ETH/DEC19" should be:
      | best static bid price | static mid price | best static offer price |
      | 141                   | 150              | 160                     |
    # LP price range is now [0.5*150.5,1.5*150.5]=[75.2,225.7]= (after ceil/floor) [76,225]
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 400    |
      | sell | 160   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 1185   |
      | buy  | 75    | 0      |

    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux2  | bestOffer | 155   | 0          | TIF_GTC |
    Then the market data for the market "ETH/DEC19" should be:
      | best static bid price | static mid price | best static offer price |
      | 141                   | 148              | 155                     |
    # LP price range is now [0.5*148,1.5*148]=[74,222]
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 0      |
      | sell | 222   | 406    |
      | sell | 160   | 0      |
      | sell | 155   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 0      |
      | buy  | 75    | 0      |
      | buy  | 74    | 1217   |

    # now decrease the minimum probability of trading
    When the following network parameters are set:
      | name                                                   | value |
      | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.05  |
    And the network moves ahead "10" blocks
    # again, this parameter is no longer meant to have any effect on order sizes
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 0      |
      | sell | 222   | 406    |
      | sell | 160   | 0      |
      | sell | 155   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 0      |
      | buy  | 75    | 0      |
      | buy  | 74    | 1217   |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 1                | TYPE_LIMIT | TIF_GTC |           |

    # trade doesn't change any of the pegs so orderbook composition is unaffected
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 225   | 0      |
      | sell | 222   | 406    |
      | sell | 160   | 0      |
      | sell | 155   | 10     |
      | buy  | 141   | 10     |
      | buy  | 140   | 0      |
      | buy  | 76    | 0      |
      | buy  | 75    | 0      |
      | buy  | 74    | 1217   |