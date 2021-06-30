Feature: Price monitoring test for issue 2668

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring updated every "1" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 43200   | 0.9999999   | 300               |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 0.8   |
    And the markets:
      | id        | quote name | asset | maturity date        | risk model               | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 300   |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Upper bound breached
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | trader1   | ETH   | 10000000000  |
      | trader2   | ETH   | 10000000000  |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 5670000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 5670000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "5670000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 4850000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 4850000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "4850000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 6630000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 6630000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 6640000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 6640000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 10min (5 min opening auction + 5 min price monitoring)
    Then time is updated to "2020-10-16T00:10:00Z"

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 15min01s (opening auction, price monitoring + extension due to time update + another period)
    Then time is updated to "2020-10-16T00:15:01Z"

    # the order was GTC, so after the auction this trade can now happen
    And the mark price should be "6640000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

  Scenario: Lower bound breached
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | trader1   | ETH   | 10000000000  |
      | trader2   | ETH   | 10000000000  |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 5670000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 5670000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "5670000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 4850000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 4850000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "4850000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 6630000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 6630000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 4840000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 4840000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 15min01s
    Then time is updated to "2020-10-16T00:15:01Z"

    And the mark price should be "4840000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

  Scenario: Upper bound breached (scale prices down by 10000)
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | trader1   | ETH   | 10000000000  |
      | trader2   | ETH   | 10000000000  |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 567      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 567      | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 567   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 567   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "567" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 485   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 485   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "485" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 663   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 663   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "663" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 664   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 664   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "663" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the mark price should be "663" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 15min01s
    Then time is updated to "2020-10-16T00:15:01Z"

    And the mark price should be "664" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
