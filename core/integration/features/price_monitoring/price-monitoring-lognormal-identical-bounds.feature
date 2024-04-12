Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99460,100541], [98999,101008])

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 10     | 0.95         | 180               |
      | 10     | 0.95         | 180                |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 60               | default-none | my-price-monitoring | default-eth-for-future | 0.01                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 60    |
      | limits.markets.maxPeggedOrders | 2     |

  Scenario: Persistent order breaks a repeated trigger (upper bound)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |

    When the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux2  | ETH/DEC21 | buy  | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC21 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "110000" for the market "ETH/DEC21"
    

    When time is updated to "2020-10-16T00:02:00Z"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | extension trigger           | mark price | indicative price | indicative volume | horizon | ref price | min bound | max bound |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 110000     | 0                | 0                 | 10      | 110000    | 109758    | 110242    |

    #T1 = T0 + 1min10s
    When time is updated to "2020-10-16T00:03:10Z"
    And the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       | extension trigger           | mark price | indicative price | indicative volume | auction end |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_UNSPECIFIED | 110000     | 111000           | 1                 | 180         |

    #T1 + 03min00s (last second of the auction)
    When time is updated to "2020-10-16T00:06:10Z"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       | extension trigger           | mark price | indicative price | indicative volume | auction end |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_UNSPECIFIED | 110000     | 111000           | 1                 | 180         |

    #T1 + 03min01s (auction doesn't get extended as the other trigger expired: last reference price was before auction start - trigger horizon)
    When time is updated to "2020-10-16T00:06:11Z"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | extension trigger           | mark price | indicative price | indicative volume | auction end | horizon | ref price | min bound | max bound |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 111000     | 0                | 0                 | 0           | 10      | 111000    | 110756    | 111245    |


