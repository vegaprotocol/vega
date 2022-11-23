
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

  Scenario: 001, If the BID & ASK reference prices are outside the valid price range the order buy order should get placed at the BID reference price and sell order should get placed at the ask reference price (0038-OLIQ-010)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |

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

  Scenario: 002, If MID reference price is above the max valid price bound then the sell order should get placed one tick above it, if the adjusted buy order is below the min price bound it should get placed at that bound, otherwise it should remain unaffected (0038-OLIQ-xxx)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 20     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 20     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 10     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 9      | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 5      | submission |

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

    # MID price is (140+180)/2=160, 
    # LP sell order should be at 160+20=180, but since reference price itself is above the max bound it should be placed at 161 (reference + minimum valid offset)
    # 1st LP buy order should be at 160-20=140, but since that adjusted price would fall below the minimum bound of 149, it gets placed at that bound
    # 2nd LP buy order should be at 160-10=150, since that's above the min bound it should be unaffected
    # 3rd LP buy order should be at 160-9=151, since that's above the min bound it should be unaffected
    # 4th LP buy order should be at 160-5=155, since that's above the min bound it should be unaffected 
    # TODO: We seem to get 159 for the case above instead
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price |  volume |
      | sell | 180   |   10    |
      | sell | 161   | 1119    |
      | buy  | 159   |    0    |
      | buy  | 151   |  299    |
      | buy  | 150   |  300    |
      | buy  | 155   |  300    |
      | buy  | 149   |  303    |
      | buy  | 140   |   10    |
      | buy  | 9     |    0    |

  Scenario: 003, If MID reference price is below the min valid price bound then the buy order should get placed one tick below it, if the adjusted sell order is above the max price bound it should get placed at that bound, otherwise it should remain unaffected (0038-OLIQ-xxx)
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                 |
      | aux   | USD   | 1000000000000          |
      | aux2  | USD   | 1000000000000          |
      | lp    | USD   | 1000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 20     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         |  1     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         |  9     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 10     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 15     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 125   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux2  | ETH/DEC19 | sell | 10     | 155   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 100     | 149       | 151       |

    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |

    # MID price is (125+155)/2=140, 
    # LP buy order should be at 140-20=120, but since reference price itself is below the min bound it should be placed at 139 (reference - minimum valid offset)
    # 1st LP sell order should be at 140+1=141, since that's below the max bound it should be unaffected
    # 2nd LP sell order should be at 140+9=149, since that's below the max bound it should be unaffected
    # 3rd LP sell order should be at 140+10=150, since that's below the max bound it should be unaffected
    # 4th LP sell order should be at 140+15=155, but since that adjusted price would fall above the maximum bound of 151, it gets placed at that bound
    # TODO: we seem to be getting 151 for the case above instead
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 139   | 1295   |
      | buy  | 125   | 10     |
      | sell | 159   | 0      |
      | sell | 155   | 10     |
      | sell | 151   | 1209   |
      | sell | 150   | 1209   |
      | sell | 149   | 303    |
      | sell | 141   | 320    |