Feature: Ensure price bounds are triggered as and when they should be, considering rounding and decimal places

  Background:
    Given the price monitoring named "st-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 10                |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
    And the log normal risk model named "st-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00011407711613050422 | 0  | 0.016 | 1.5   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config     | decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | st-log-normal-risk-model | default-margin-calculator | 1                | default-none | st-price-monitoring | default-eth-for-future | 6              | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  @STAuc
  Scenario: Replicate issue where price monitoring should be triggered at min bound - 1
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount                        |
      | party1  | ETH   | 10000000000000000000000000000 |
      | party2  | ETH   | 10000000000000000000000000000 |
      | party3  | ETH   | 10000000000000000000000000000 |
      | party4  | ETH   | 10000000000000000000000000000 |
      | partyLP | ETH   | 10000000000000000000000000000 |
      | aux     | ETH   | 10000000000000000000000000000 |

    # 977142641
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     |
      | party3 | ETH/DEC20 | buy  | 1      | 977142640  | 0                | TYPE_LIMIT | TIF_GFA |
      | party2 | ETH/DEC20 | sell | 1      | 977142640  | 0                | TYPE_LIMIT | TIF_GFA |
      | party1 | ETH/DEC20 | buy  | 5      | 950000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 5      | 1070000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | buy  | 1      | 950000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 1070000000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount       | fee | side | pegged reference | proportion | offset  | lp type    |
      | lp1 | party1 | ETH/DEC20 | 39050000000000000000000 | 0.3 | buy  | BID              | 2          | 1000000 | submission |
      | lp1 | party1 | ETH/DEC20 | 39050000000000000000000 | 0.3 | sell | ASK              | 13         | 1000000 | submission |
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "977142640" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     |
      | aux    | ETH/DEC20 | buy  | 1      | 977142641 | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC20 | sell | 1      | 977142641 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 977142640  | 977142641         | TRADING_MODE_CONTINUOUS |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake           | supplied stake          | open interest |
      | 977142640  | TRADING_MODE_CONTINUOUS | 5       | 975999651 | 978286619 | 1080524331312000000000 | 39050000000000000000000 | 2             |

    When the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     |
      | aux    | ETH/DEC20 | buy  | 1      | 975999650 | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC20 | sell | 1      | 975999650 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode                    | auction trigger       | target stake           | supplied stake          | open interest |
      | 977142640  | 977142641         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1618890619455000000000 | 39050000000000000000000 | 2             |

  @STAuc
  Scenario: Replicate  issue where price bounds are violated by 1 * 10^(market decimal places)
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount                        |
      | party1  | ETH   | 10000000000000000000000000000 |
      | party2  | ETH   | 10000000000000000000000000000 |
      | party3  | ETH   | 10000000000000000000000000000 |
      | party4  | ETH   | 10000000000000000000000000000 |
      | partyLP | ETH   | 10000000000000000000000000000 |
      | aux     | ETH   | 10000000000000000000000000000 |

    # 977142641
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     |
      | party3 | ETH/DEC20 | buy  | 1      | 977142640  | 0                | TYPE_LIMIT | TIF_GFA |
      | party2 | ETH/DEC20 | sell | 1      | 977142640  | 0                | TYPE_LIMIT | TIF_GFA |
      | party1 | ETH/DEC20 | buy  | 5      | 950000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 5      | 1070000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | buy  | 1      | 950000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 1070000000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount       | fee | side | pegged reference | proportion | offset  | lp type    |
      | lp1 | party1 | ETH/DEC20 | 39050000000000000000000 | 0.3 | buy  | BID              | 2          | 1000000 | submission |
      | lp1 | party1 | ETH/DEC20 | 39050000000000000000000 | 0.3 | sell | ASK              | 13         | 1000000 | submission |
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "977142640" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     |
      | aux    | ETH/DEC20 | buy  | 1      | 977142641 | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC20 | sell | 1      | 977142641 | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake           | supplied stake          | open interest |
      | 977142640  | 977142641         | TRADING_MODE_CONTINUOUS | 5       | 975999651 | 978286619 | 1080524331312000000000 | 39050000000000000000000 | 2             |

    When the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     |
      | aux    | ETH/DEC20 | buy  | 1      | 974999651 | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC20 | sell | 1      | 974999651 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode                    | auction trigger       | target stake           | supplied stake          | open interest |
      | 977142640  | 977142641         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1617231921113700000000 | 39050000000000000000000 | 2             |
