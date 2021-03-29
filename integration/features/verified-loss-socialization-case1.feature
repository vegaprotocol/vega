Feature: Test loss socialization case 1

  Background:
    Given the insurance pool initial balance for the markets is "0":

    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: case 1 from https://docs.google.com/spreadsheets/d/1CIPH0aQmIKj6YeFW9ApP_l-jwB4OcsNQ/edit#gid=1555964910
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader           | asset | amount    |
      | sellSideProvider | BTC   | 100000000 |
      | buySideProvider  | BTC   | 100000000 |
      | trader1          | BTC   | 5000      |
      | trader2          | BTC   | 50000     |
      | trader3          | BTC   | 50000     |
      | aux1             | BTC   | 100000000 |
      | aux2             | BTC   | 100000000 |
# setup orderbook
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 120   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux1             | ETH/DEC19 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux2             | ETH/DEC19 | buy  | 1      | 80    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
      | aux1             | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
# trader 1 place an order + we check margins
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
# then trader2 place an order, and we calculate the margins again
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 100    | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
# then we change the volume in the book
    Then traders cancel the following orders:
      | trader           | reference       |
      | sellSideProvider | sell-provider-1 |
      | buySideProvider  | buy-provider-1  |
    Then the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    When traders place the following orders:
      | trader           | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 80    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |
    Then traders cancel the following orders:
      | trader | reference |
      | aux1   | aux-s-1   |
      | aux2   | aux-b-1   |
    Then the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 100    | 180   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader3 | ETH/DEC19 | sell | 100    | 180   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    Then traders have the following profit and loss:
      | trader  | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | -5000        |
      | trader2 | 200    | 8000           | -2970        |
      | trader3 | -100   | 0              | 0            |
    And the insurance pool balance is "0" for the market "ETH/DEC19"
    And Cumulated balance for all accounts is worth "400105000"
