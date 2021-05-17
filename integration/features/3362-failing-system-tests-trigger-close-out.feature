Feature: Replicate failing system tests after changes to price monitoring (not triggering with FOK orders, auction duration)

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring updated every "4" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | maturity date        | risk model               | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 1      |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Replicate test called test_TriggerWithMarketOrder
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount    |
      | trader1  | ETH   | 100000000 |
      | trader2  | ETH   | 100000000 |
      | trader3  | ETH   | 100000000 |
      | trader4  | ETH   |    800000 |
      | traderLP | ETH   | 100000000 |
      | aux      | ETH   | 100000000 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader1 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GFA |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GFA |
      | trader1 | ETH/DEC20 | buy  | 5      | 95000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 5      | 107000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 95000  | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 107000 | 0                | TYPE_LIMIT  | TIF_GTC |
    And the traders submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader1  | ETH/DEC20 | 16000000          | 0.3 | buy        | BID             | 2                | -10          |
      | lp1 | trader1  | ETH/DEC20 | 16000000          | 0.3 | sell       | ASK             | 13               | 10           |
    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    Then debug market data for "ETH/DEC20"
    When the opening auction period ends for market "ETH/DEC20"
    Then the mark price should be "100000" for the market "ETH/DEC20"


    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    ## price bounds are 99711 - 99845 - 100156 - 100290
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader2 | ETH/DEC20 | sell | 15     | 107500 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 10     | 107100 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader3 | ETH/DEC20 | buy  | 10     | 107300 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 10     | 107100 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    ## All price bounds were exceeded -> duration = 14s + 2 opening auction = 16s (+1 to ensure auction ends)
    ## New price bounds are: 106790 - 106933 - 107267 - 107410 (outer 8s, inner 6s)
    When time is updated to "2020-10-16T00:00:17Z"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "107100" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader4 | ETH/DEC20 | buy  | 50     | 107500 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "107100" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader3 | ETH/DEC20 | buy  | 70     | 106000 | 0                | TYPE_LIMIT  | TIF_GFA |
    And the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset |
      | trader4 | ETH/DEC20 | buy  | 35     | BID       | -1000  |
      | trader4 | ETH/DEC20 | sell | 35     | ASK       | 3000   |
    And the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | trader2 | ETH/DEC20 | sell | 80     | 105000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | trader3 | ETH/DEC20 | buy  | 81     | 106000 | 0                | TYPE_LIMIT  | TIF_GFA |
      | trader3 | ETH/DEC20 | buy  | 86     | 107000 | 0                | TYPE_LIMIT  | TIF_GTC |
    And the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset |
      | trader1 | ETH/DEC20 | buy  | 100    | BID       | -5000  |
      | trader2 | ETH/DEC20 | sell | 95     | ASK       | 1000   |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the mark price should be "107100" for the market "ETH/DEC20"

    ## Now we can end the auction (time was T+17, auction triggered by 2 bound violations -> + 14s + 1s to ensure we end)
    ## means T+32s
    When time is updated to "2020-10-16T00:00:32Z"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "106000" for the market "ETH/DEC20"
    ## When time is updated to "2020-10-16T00:00:33Z"
    ## Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    ## This isn't working ATM
    ## And the traders should have the following account balances:
    ##  | trader  | asset | market id | margin | general | bond |
    ##  | trader4 | ETH   | ETH/DEC20 |      0 |       0 |    0 |
