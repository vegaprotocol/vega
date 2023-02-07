Feature: Closeout LP scenarios with a trader comes with a crazy order
  # Replicate a scenario from Lewis
  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the price monitoring named "price-monitoring-1":
      | horizon  | probability | auction extension |
      | 72000000 | 0.99        | 3                 |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 3              |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring   | data source config     | decimal places | position decimal places |
      | ETH/DEC20 | ETH        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | price-monitoring-1 | default-eth-for-future | 3              | 0                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.liquidity.stakeToCcyVolume       | 1     |

  Scenario: Replicate a scenario from Lewis
    # 1. trader B made LP commitment 150,000
    # 2. trader C and A cross at 0.5 with size of 111, and this opens continuous trading (trade B is short)
    # 3. trader C comes with an order with crazy price
    # 4. trader B’s margin has increased sharply because of the order (from step2),
    # 5. trader A and C and trigger MTM
    # 6. trader B got closeout out, and the closeout trade was between trader B - network - trader C

    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount         |
      | traderA | USD   | 10000000000000 |
      | traderB | USD   | 3100000        |
      | traderC | USD   | 10000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | traderB | ETH/DEC20 | 150000            | 0.001 | sell | ASK              | 100        | 20     | submission |
      | lp1 | traderB | ETH/DEC20 | 150000            | 0.001 | buy  | BID              | 100        | 20     | amendmend  |
      | lp2 | traderC | ETH/DEC20 | 15                | 0.001 | sell | ASK              | 100        | 20     | submission |
      | lp2 | traderC | ETH/DEC20 | 15                | 0.001 | buy  | BID              | 100        | 20     | amendmend  |

    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | traderA | ETH/DEC20 | buy  | 1      | 49    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | traderB | ETH/DEC20 | sell | 1      | 350   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | traderA | ETH/DEC20 | buy  | 1      | 350   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | traderB | ETH/DEC20 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
      | traderB | ETH/DEC20 | sell | 1      | 3000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC20"

    And the parties should have the following account balances:
      | party   | asset | market id | margin  | general | bond   |
      | traderB | USD   | ETH/DEC20 | 2899518 | 50482   | 150000 |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | traderA | 350   | 1    | traderB |

    And the market data for the market "ETH/DEC20" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 12449        | 150015         | 1             |

    Then the order book should have the following volumes for market "ETH/DEC20":
      | side | price | volume |
      | buy  | 29    | 5174   |
      | buy  | 49    | 1      |
      | sell | 2000  | 1      |
      | sell | 2020  | 75     |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | traderA | ETH/DEC20 | buy  | 111    | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | traderB | ETH/DEC20 | sell | 111    | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond   |
      | traderB | USD   | ETH/DEC20 | 511138 | 2439156 | 150000 |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 50         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 199186       | 150015         | 112           |

    # When the parties submit the following liquidity provision:
    #   | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
    #   | lp2 | traderC | ETH/DEC20 | 150000            | 0.001 | sell | ASK              | 100        | 20     | amendmend |
    #   | lp2 | traderC | ETH/DEC20 | 150000            | 0.001 | buy  | BID              | 100        | 20     | amendmend |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price       | resulting trades | type       | tif     |
      | traderC | ETH/DEC20 | sell | 120    | 45000000000 | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond |
      | traderB | USD   | ETH/DEC20 | 0      | 0       | 0    |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 50         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 199186       | 15             | 112           |

    And the following trades should be executed:
      | buyer   | price       | size | seller  |
      | network | 45000000000 | 112  | traderC |
      | traderB | 45000000000 | 112  | network |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl   |
      | traderA | 112    | -300           | 0              |
      | traderB | 0      | 0              | -3099994       |
      | traderC | -112   | 5039999994400  | -5039999994400 |

    And the insurance pool balance should be "0" for the market "ETH/DEC20"


