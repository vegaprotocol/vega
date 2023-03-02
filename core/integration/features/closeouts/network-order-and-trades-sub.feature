Feature: Ensure network party are generated

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e3                    | 1e3                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Implement trade and order network
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLoser  | BTC   | 12000         |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |

    # insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 100    | 159   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux              | ETH/DEC19 | sell | 1      | 149   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 149   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
      | aux2             | ETH/DEC19 | buy  | 100    | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # insurance pool generation - trade
    When the parties place the following orders "1" blocks apart:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser  | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # insurance pool generation - modify order book
    Then the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 400    | 40    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |
    And the parties cancel the following orders:
      | party | reference |
      | aux2  | aux-b-2   |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # insurance pool generation - set new mark price (and trigger closeout)
    When the parties place the following orders "1" blocks apart:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # check the network trade happened
    Then the following network trades should be executed:
      | party            | aggressor side | volume |
      | designatedLoser  | buy            | 290    |
      | buySideProvider  | sell           | 290    |
