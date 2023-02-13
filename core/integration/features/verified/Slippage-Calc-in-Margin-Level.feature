Feature: Test closeout type 1: margin >= cost of closeout

  Background:
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    # risk factor short: 0.48787313795861700

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 2             | 2.5            | 3              |

    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 100000  | 0.9999999   | 3                 |

    And the markets:
      | id        | quote name | asset | risk model             | margin calculator   | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | USD        | USD   | lognormal-risk-model-1 | margin-calculator-1 | 1                | default-none | price-monitoring | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    And the average block duration is "1"

  Scenario: case 1 test closeout cost and insurance pool balance
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount     |
      | sellSideProvider | USD   | 1000000000 |
      | buySideProvider  | USD   | 1000000000 |
      | party1           | USD   | 30000      |
      | party2           | USD   | 50000000   |
      | party3           | USD   | 50000000   |
      | aux1             | USD   | 1000000000 |
      | aux2             | USD   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC19 | 20000             | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC19 | 20000             | 0.001 | sell | ASK              | 1          | 10     | amendment  |
      | lp1 | aux1  | ETH/DEC19 | 20000             | 0.001 | buy  | MID              | 1          | 10     | amendment  |
      | lp1 | aux1  | ETH/DEC19 | 20000             | 0.001 | sell | MID              | 1          | 26     | amendment  |
    When the network moves ahead "1" blocks
    # setup order book
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux1             | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux1             | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 95    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux2             | ETH/DEC19 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |

    When the network moves ahead "1" blocks

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100        | TRADING_MODE_CONTINUOUS | 100000  | 70        | 142       | 487          | 20000          | 1             |

    # party1 place an order
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    # party1 maintenance margin should be: ordersize*markprice*riskfactor = 100*100*0.48787313795861700=4879
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 4879        | 9758   | 12197   | 14637   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 12197  | 17803   |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      #original vol
      | sell | 150   | 1000   |
      #LP pegged vol
      | sell | 123   | 81     |
      #LP pegged vol
      | sell | 110   | 91     |
      #original vol
      | sell | 105   | 1      |

    When the network moves ahead "1" blocks

    # party1 will have a position
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 100    | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      #original vol
      | sell | 150   | 1000   |
      #LP pegged vol
      | sell | 126   | 79     |
      #LP pegged vol
      | sell | 115   | 87     |
      #original vol
      | sell | 105   | 1      |

    When the network moves ahead "1" blocks
    Then the mark price should be "100" for the market "ETH/DEC19"

    # TODO: these calculations, due to MTM changes are not entirely accurate
    # slippage is calculated from the order book before the trade happens, which is (105+115*99)/100-100=14
    # party1 maintenance margin should be: position_vol* slippage + vol * riskfactor * markprice= 100*14 + 100*0.48787313795861700*100=6278.73 rounded up to 6279
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 6479        | 12958  | 16197   | 19437   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 16197  | 13803   |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 500    | 126   | 3                | TYPE_LIMIT | TIF_GTC | ref-1-xxx |

    Then the mark price should be "126" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 21370  | 6030    |



