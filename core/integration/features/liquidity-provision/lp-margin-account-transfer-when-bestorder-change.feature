
Feature:  test 0038-OLIQ-008
  Background:
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 2     |
    #rf_short=0.9247862
    #rf_long=0.499476497
    And the price monitoring named "price-monitoring-1":
      | horizon | probability       | auction extension |
      | 36000   | 0.999999999999999 | 300               |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |

    And the following network parameters are set:
      | name                                                   | value |
      | market.liquidity.minimum.probabilityOfTrading.lpOrders | 0.01  |
      | network.markPriceUpdateMaximumFrequency                | 0s    |
    And the average block duration is "1"

  Scenario: If best bid / ask has changed and the LP order volume is moved around to match the shape / new peg levels then the margin requirement for the party may change. There is at most one transfer in / out of the margin account of the LP party as a result of one of the pegs moving. 0038-OLIQ-008

    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount        |
      | aux   | USD   | 1000000000000 |
      | aux2  | USD   | 1000000000000 |
      | aux3  | USD   | 1000000000000 |
      | aux4  | USD   | 1000000000000 |
      | lp    | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0   | buy  | BID              | 50         | 10     | submission |
      | lp1 | lp    | ETH/DEC19 | 90000             | 0   | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux3  | ETH/DEC19 | buy  | 10     | 140   | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | aux4  | ETH/DEC19 | sell | 10     | 160   | 0                | TYPE_LIMIT | TIF_GTC | bestOffer |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the opening auction period ends for market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 88        | 257       |

    And the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp1 | lp    | ETH/DEC19 | 90000             | STATUS_ACTIVE |

    # LP pegged order should be replaced with respect to LP_price_range rather than price_monitoring_bounds
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 170   | 530    |
      | sell | 160   | 10     |
      | buy  | 140   | 10     |
      | buy  | 130   | 693    |

    Then the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | aux3  | ETH/DEC19 | 750         | 900    | 1125    | 1500    |
      | aux4  | ETH/DEC19 | 1388        | 1665   | 2082    | 2776    |
      | lp    | ETH/DEC19 | 73521       | 88225  | 110281  | 147042  |
    #lp_margin = max(530*150*0.9247862,693*150*0.499476491)=73521

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general      |
      | aux3  | USD   | ETH/DEC19 | 1050   | 999999998950 |
      | aux4  | USD   | ETH/DEC19 | 2220   | 999999997780 |
      | lp    | USD   | ETH/DEC19 | 110281 | 999999799719 |

    And the network moves ahead "10" blocks
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 170   | 530    |
      | sell | 160   | 10     |
      | buy  | 140   | 10     |
      | buy  | 130   | 693    |

    # update the best offer
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux4  | bestOffer | 165   | 0          | TIF_GTC |

    # observe volumes change
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 175   | 515    |
      | sell | 165   | 10     |
      | buy  | 140   | 10     |
      | buy  | 130   | 693    |

    Then the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | aux3  | ETH/DEC19 | 750         | 900    | 1125    | 1500    |
      | aux4  | ETH/DEC19 | 1388        | 1665   | 2082    | 2776    |
      | lp    | ETH/DEC19 | 71440       | 85728  | 107160  | 142880  |

    # no transfer in lp account since the existing margin is under release level, and above search level
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general      |
      | aux3  | USD   | ETH/DEC19 | 1050   | 999999998950 |
      | aux4  | USD   | ETH/DEC19 | 2220   | 999999997780 |
      | lp    | USD   | ETH/DEC19 | 110281 | 999999799719 |

    # update the best offer
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux4  | bestOffer | 220   | 0          | TIF_GTC |

    # observe volumes change
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 230   | 392    |
      | sell | 220   | 10     |
      | buy  | 140   | 10     |
      | buy  | 130   | 693    |

    Then the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | aux3  | ETH/DEC19 | 750         | 900    | 1125    | 1500    |
      | aux4  | ETH/DEC19 | 1388        | 1665   | 2082    | 2776    |
      | lp    | ETH/DEC19 | 54378       | 65253  | 81567   | 108756  |

    # transder from general to margin account since the existing margin account is above release level
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general      |
      | aux3  | USD   | ETH/DEC19 | 1050   | 999999998950 |
      | aux4  | USD   | ETH/DEC19 | 2220   | 999999997780 |
      | lp    | USD   | ETH/DEC19 | 81567  | 999999828433 |