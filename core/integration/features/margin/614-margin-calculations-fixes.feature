Feature: test bugfix 614 for margin calculations

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @MTMDelta
  Scenario: CASE-1: Trader submits long order that will trade - new formula & high exit price
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount  |
      | chris   | ETH   | 10000   |
      | edd     | ETH   | 10000   |
      | barney  | ETH   | 10000   |
      | rebecca | ETH   | 10000   |
      | tamlyn  | ETH   | 10000   |
      | party1  | ETH   | 1000000 |
      | party2  | ETH   | 1000000 |
      | aux     | ETH   | 1000    |
      | lpprov  | ETH   | 1000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 87    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 250   | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | chris   | ETH/DEC19 | sell | 100    | 250   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | edd     | ETH/DEC19 | sell | 11     | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | barney  | ETH/DEC19 | sell | 2      | 112   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | barney  | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | edd     | ETH/DEC19 | buy  | 3      | 96    | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | chris   | ETH/DEC19 | buy  | 15     | 90    | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | rebecca | ETH/DEC19 | buy  | 50     | 87    | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
    # this is now the actual party that we are testing
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tamlyn | ETH/DEC19 | buy  | 13     | 150   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | tamlyn | ETH/DEC19 | 988         | 3161   | 3952    | 4940    |
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | tamlyn  | ETH   | ETH/DEC19 | 3952   | 6104    |
      | chris   | ETH   | ETH/DEC19 | 5600   | 4400    |
      | edd     | ETH   | ETH/DEC19 | 5456   | 4544    |
      | barney  | ETH   | ETH/DEC19 | 992    | 8952    |
      | rebecca | ETH   | ETH/DEC19 | 5600   | 4400    |
    And the cumulated balance for all accounts should be worth "3051000"
