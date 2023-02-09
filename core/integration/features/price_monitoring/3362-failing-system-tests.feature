Feature: Replicate failing system tests after changes to price monitoring (not triggering with FOK orders, auction duration)

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | 
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Replicate test called test_TriggerWithMarketOrder
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | party1  | ETH   | 100000000 |
      | party2  | ETH   | 100000000 |
      | party3  | ETH   | 100000000 |
      | partyLP | ETH   | 100000000 |
      | aux     | ETH   | 100000000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA |
      | party2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA |
      | party1 | ETH/DEC20 | buy  | 5      | 95000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 5      | 107000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | buy  | 1      | 95000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 107000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC20 | 16000000          | 0.3 | buy  | BID              | 2          | 10     | submission |
      | lp1 | party1 | ETH/DEC20 | 16000000          | 0.3 | sell | ASK              | 13         | 10     | amendment  |
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    ## price bounds are 99771 to 100290
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 1      | 100150 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 100150 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100000     | 100150            | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 1      | 99845 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | buy  | 2      | 99844 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    # Now place a FOK order that would trigger a price auction (party 1 has a buy at 95,000 on the book

    And the order book should have the following volumes for market "ETH/DEC20":
      | side | price  | volume |
      | sell | 107010 | 150    |
      | sell | 107000 | 6      |
      | buy  | 99845  | 1      |
      | buy  | 99844  | 2      |
      | buy  | 99835  | 152    |
      | buy  | 95000  | 6      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | error                                                       |
      | party2 | ETH/DEC20 | sell | 156    | 0     | 0                | TYPE_MARKET | TIF_FOK | OrderError: non-persistent order trades out of price bounds |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound |
      | 100000     | 100150            | TRADING_MODE_CONTINUOUS | 5       | 99845     | 100156    |
      | 100000     | 100150            | TRADING_MODE_CONTINUOUS | 10      | 99711     | 100290    |

    # Now set the volume so that the order generates a trade that's still within price monitoring bounds
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party2 | ETH/DEC20 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_FOK |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the network moves ahead "10" blocks

    And the mark price should be "99845" for the market "ETH/DEC20"