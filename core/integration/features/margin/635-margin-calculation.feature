Feature: Regression test for issue 596

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @ignore
  Scenario: Traded out position but monies left in margin account
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount     |
      | partyGuy         | BTC   | 1000000000 |
      | sellSideProvider | BTC   | 1000000000 |
      | buySideProvider  | BTC   | 1000000000 |
      | aux              | BTC   | 1000000000 |
      | aux2             | BTC   | 1000000000 |
      | lpprov           | BTC   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price    | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 8700000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 25000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"


    # setup previous mark price
    Then the parties place the following orders with ticks:
      | party            | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 10300000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 10300000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 100    | 25000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | sellSideProvider | ETH/DEC19 | sell | 11     | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | sellSideProvider | ETH/DEC19 | sell | 2      | 11200000 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | buySideProvider  | ETH/DEC19 | buy  | 3      | 9600000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | buySideProvider  | ETH/DEC19 | buy  | 15     | 9000000  | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | buySideProvider  | ETH/DEC19 | buy  | 50     | 8700000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
    # buy 13@150
    Then the parties place the following orders with ticks:
      | party    | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 13     | 15000000 | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
    # checking margins
    Then the parties should have the following account balances:
      | party    | asset | market id | margin    | general   |
      | partyGuy | BTC   | ETH/DEC19 | 394400032 | 611199968 |
    # checking margins levels
    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search    | initial   | release   |
      | partyGuy | ETH/DEC19 | 98600008    | 315520025 | 394400032 | 493000040 |
