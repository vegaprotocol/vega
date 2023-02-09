Feature: test AC 006-POSI-009, 006-POSI-027

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario:  Open long position, trades occur closing the long position and opening a short position (006-POSI-009; 006-POSI-027)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000    |
      | party2 | ETH   | 10000    |
      | party3 | ETH   | 10000    |
      | aux    | ETH   | 100000   |
      | aux2   | ETH   | 100000   |
      | aux3   | ETH   | 100000   |
      | lpprov | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 5      | 49    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 5      | 5001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 4921   | 5079    |
      | party2 | ETH   | ETH/DEC19 | 1273   | 8627    |

    # party3 does not have position record exist since party3 does not have either an open position nor active order
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | -1     | 0              | 0            |
      | aux2   | 1      | 0              | 0            |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 5041   | 4959    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 7682   | 1318    |
      | party2 | ETH   | ETH/DEC19 | 2605   | 8295    |
      | party3 | ETH   | ETH/DEC19 | 2605   | 7195    |

    Then the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000   | ETH   |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000   | ETH   |
    And the cumulated balance for all accounts should be worth "10330000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | -1     | -1000          | 0            |
      | aux2   | 1      | 1000           | 0            |
      | party1 | -2     | -1000          | 0            |
      | party2 | 1      | 1000           | 0            |
      | party3 | 1      | 0              | 0            |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC19 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | -1     | 0              | 0            |
      #aux2 closed long position: AC 0006-POSI-009
      | aux2   | 0      | 0              | 0            |
      #aux3 opened short position
      | aux3   | 1      | 0              | 0            |
      | party1 | -2     | 1000           | 0            |
      | party2 | 1      | 0              | 0            |
      | party3 | 1      | -1000          | 0            |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS |         |           |           | 3300         | 9000           | 3             |


