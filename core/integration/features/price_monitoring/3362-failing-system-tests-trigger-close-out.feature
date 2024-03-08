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
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 0.01                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | limits.markets.maxPeggedOrders          | 1500  |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Replicate test called test_TriggerWithMarketOrder
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | party1  | ETH   | 100000000 |
      | party2  | ETH   | 100000000 |
      | party3  | ETH   | 100000000 |
      | party4  | ETH   | 800000    |
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
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/DEC20 | 16000000          | 0.3 | submission |
      | lp1 | party1 | ETH/DEC20 | 16000000          | 0.3 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | party1 | ETH/DEC20 | 2         | 1                    | buy  | BID              | 2          | 10     |
      | party1 | ETH/DEC20 | 2         | 1                    | sell | ASK              | 13         | 10     |
 
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "100000" for the market "ETH/DEC20"


    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    ## price bounds are 99711 - 99845 - 100156 - 100290
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party2 | ETH/DEC20 | sell | 15     | 107500 | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | buy  | 10     | 107100 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party3 | ETH/DEC20 | buy  | 10     | 107300 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 10     | 107100 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When time is updated to "2020-10-16T00:00:09Z"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | extension trigger     | auction end |
      | 100000     | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_PRICE | 14          |
    When time is updated to "2020-10-16T00:00:18Z"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "107100" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party4 | ETH/DEC20 | buy  | 50     | 107500 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "107100" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party3 | ETH/DEC20 | buy  | 70     | 106000 | 0                | TYPE_LIMIT | TIF_GFA |
    And the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party4 | ETH/DEC20 | buy  | 35     | BID              | 1000   |
      | party4 | ETH/DEC20 | sell | 35     | ASK              | 3000   |
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     |
      | party2 | ETH/DEC20 | sell | 80     | 105000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC20 | buy  | 81     | 106000 | 0                | TYPE_LIMIT | TIF_GFA |
      | party3 | ETH/DEC20 | buy  | 86     | 107000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party1 | ETH/DEC20 | buy  | 100    | BID              | 5000   |
      | party2 | ETH/DEC20 | sell | 95     | ASK              | 1000   |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "107100" for the market "ETH/DEC20"

    # Need to update time twice so that 2nd bound gets activated when attempting to leave the auction after first trigger extension elapses
    When time is updated to "2020-10-16T00:00:25Z"
    When time is updated to "2020-10-16T00:00:33Z"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "106000" for the market "ETH/DEC20"
