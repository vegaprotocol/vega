Feature: Amend order to trigger price monitoring auction

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the markets:
      | id        | quote name | asset | risk model             | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | system-test-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 6     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Upper bound breached
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount       |
      | party1    | ETH   | 100000000000 |
      | party2    | ETH   | 100000000000 |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |
      | lpprov    | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 5670000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 10     | 5670010 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the mark price should be "5670000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties amend the following orders:
      | party  | reference | price   | size delta | tif     |
      | party2 | ref-2     | 5670005 | 0          | TIF_GTC |
    Then the mark price should be "5670000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
