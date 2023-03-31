Feature: Position resolution case 5 lognormal risk model

  Background:
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    #calculated risk factor long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999999  | 300               |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future | 2                       | 1e-3                   | 1e-3                      |

    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: using lognormal risk model, set "designatedLoser " closeout while the position of "designatedLoser " is not fully covered by orders on the order book

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | USD   | 1000000000000 |
      | buySideProvider  | USD   | 1000000000000 |
      | designatedLoser  | USD   | 32000         |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lpprov           | USD   | 1000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1000   | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1000   | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 100    | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 100    | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2001  | 450    |
      | sell | 2000  | 1000   |
      | buy  | 1     | 901000 |

    # insurance pool generation - setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 29000  | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 100    | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2000  | 1000   |
      | sell | 250   | 3600   |
      | sell | 150   | 29000  |
      | buy  | 140   | 100    |
      | buy  | 40    | 22500  |
      | buy  | 1     | 1000   |

    # insurance pool generation - trade
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser  | ETH/DEC19 | buy  | 29000  | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLoser  | USD   | ETH/DEC19 | 27650  | 0       |

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLoser  | 29000  | 0              | 0            |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 2100  | 429    |
      | sell | 2000  | 1000   |
      | buy  | 140   | 100    |
      | buy  | 40    | 22500  |
      | buy  | 1     | 1000   |

    Then the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |

    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-4     |

    And the mark price should be "140" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedLoser  | USD   | ETH/DEC19 | 0      | 0       |

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | designatedLoser  | 0      | 0              | -27650       |

    # then we make sure the insurance pool collected the funds (however they get later spent on MTM payment to closeout-facilitating party)
    Then the following transfers should happen:
      | from             | to              | from account            | to account                       | market id | amount | asset |
      | designatedLoser  | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 0      | USD   |
      | buySideProvider  | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 0      | USD   |
      | buySideProvider  | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 | 1      | USD   |
      | designatedLoser  |                 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC19 | 0      | USD   |
      | market           | lpprov          | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 | 0      | USD   |
      | designatedLoser  | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_INSURANCE           | ETH/DEC19 | 24721  | USD   |
      | market           | market          | ACCOUNT_TYPE_INSURANCE  | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC19 | 24721  | USD   |
      | market           | lpprov          | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 23869  | USD   |
      | buySideProvider  | buySideProvider | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 76     | USD   |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"
