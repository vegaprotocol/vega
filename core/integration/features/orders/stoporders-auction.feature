Feature: stop orders

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC20 | BTC        | BTC   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future | 1e-3                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  Scenario: Submit a trailing order during opening auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 10     | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | only   | fb trailing | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | 0.1         | tstop     |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # Check the mark price is correct
    And the mark price should be "50" for the market "ETH/DEC19"

    # Now we make sure the trailing stop is working correctly
    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING   | tstop     |

    # Move the mark price down by <10% to not trigger the orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC19 | buy  | 1      | 48    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 48    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the stop orders should have the following states
      | party  | market id | status         | reference |
      | party1 | ETH/DEC19 | STATUS_PENDING | tstop     |

    # Move the mark price down by 10% to trigger the orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 40    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the stop orders should have the following states
      | party  | market id | status           | reference |
      | party1 | ETH/DEC19 | STATUS_TRIGGERED | tstop     |



