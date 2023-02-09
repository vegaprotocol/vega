Feature: Regression test for issue 630

  Background:

    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Trader is being closed out.
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount  |
      | sellSideProvider | BTC   | 1000000 |
      | buySideProvider  | BTC   | 1000000 |
      | partyGuy         | BTC   | 240000  |
      | party1           | BTC   | 1000000 |
      | party2           | BTC   | 1000000 |
      | aux              | BTC   | 100000  |
      | lpprov           | BTC   | 1000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

    # setup orderbook
    When the parties place the following orders "1" blocks apart:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 200    | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 200    | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the cumulated balance for all accounts should be worth "5340000"
    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | sellSideProvider | ETH/DEC19 | 2000        | 2200   | 2400    | 2800    |
    When the parties place the following orders "1" blocks apart:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | partyGuy         | BTC   | ETH/DEC19 | 0      | 0       |
      | sellSideProvider | BTC   | ETH/DEC19 | 240000 | 760000  |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "5340000"
