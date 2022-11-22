
Feature: Confirm automatic adjustments to LP orders when reference price is out of valid price ranges as specified in 0038-OLIQ

  Background:
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 2     |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99        | 300               |
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

  Scenario: 001, If the reference price itself is outside the valid price range the order should get placed at - when bid/ask is used as a reference (0038-OLIQ-010)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 140   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux2  | ETH/DEC19 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 100     | 149       | 151       |

    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2465  | 0      |
      | sell | 160   | 1135   |
      | buy  | 140   | 1296   |
      | buy  | 9     | 0      |

  Scenario: 002, If the reference price itself is outside the valid price range (MID above max valid price) the order should get placed at - one tick away from it - when mid is used as a reference. (0038-OLIQ-010)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 20     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 20     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 140   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux2  | ETH/DEC19 | sell | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 100     | 149       | 151       |

    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |

    # MID price is (140+180)/2=160, so expecting LP buy orders at 159 and LP sell orders at 161
    # TODO: The buy orders get placed at at the min bound though
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2465  | 0      |
      | sell | 180   | 10     |
      | sell | 161   | 1119   |
      | buy  | 159   | 0      |
      | buy  | 149   | 1209   |
      | buy  | 140   | 10     |
      | buy  | 9     | 0      |

Scenario: 002, If the reference price itself is outside the valid price range (MID below min valid price) the order should get placed at - one tick away from it - when mid is used as a reference. (0038-OLIQ-010)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 20     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 20     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 139   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux2  | ETH/DEC19 | sell | 10     | 151   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 100     | 149       | 151       |

    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |

    # MID price is (139+151)/2=145, so expecting LP buy orders at 144 and LP sell orders at 146
    # TODO: The sell orders get placed at at the max bound though
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 151   | 1203   |
      | sell | 145   | 0      |
      | buy  | 144   | 1250   |
      | buy  | 139   | 10     |

