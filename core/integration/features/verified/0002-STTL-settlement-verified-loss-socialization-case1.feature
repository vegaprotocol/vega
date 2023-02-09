Feature: Test loss socialization case 1

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Case 1: trader1 has insufficient MTM & only trader2 socialises the losses (0002-STTL-009)
    Description : Case 1 from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | party1           | BTC   | 5000      |
      | party2           | BTC   | 50000     |
      | party3           | BTC   | 50000     |
      | aux1             | BTC   | 100000000 |
      | aux2             | BTC   | 100000000 |
    # setup order book
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux1             | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
      | aux1             | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # party 1 place an order + we check margins
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # then party2 place an order, and we calculate the margins again
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 100    | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # then we change the volume in the book
    Then the parties cancel the following orders:
      | party            | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |
    Then the parties cancel the following orders:
      | party | reference |
      | aux1  | aux-s-1   |
      | aux2  | aux-b-1   |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 100    | 180   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party3 | ETH/DEC19 | sell | 100    | 180   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | -5000        |
      | party2 | 200    | 8000           | -2970        |
      | party3 | -100   | 0              | 0            |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "400105000"
