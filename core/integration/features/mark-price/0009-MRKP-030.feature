Feature: Test setting of mark price
  Background:
    Given the average block duration is "1"
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | median     | 0.5          | 2           | 0           | 0,1,0,0        | 6s,4s,24h0m0s,1h25m0s      |

  @SLABug
  Scenario: 001 check mark price using order price with cash amount 100 USD
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party            | ETH/FEB23 | sell | 2      | 15902  | 0                | TYPE_LIMIT | TIF_GTC |
      | party            | ETH/FEB23 | sell | 1      | 15904  | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |

    #AC 0009-MRKP-030
    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/FEB23"

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  |
      | party | ETH/FEB23 | 9546        | 10500  | 11455   | 13364   | cross margin |
    #margin = min(3*(15900-15900), 3*15900*(0.25))+3*0.1*15900 + 3*0.1*15900=9546

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 11449  | 36601   |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    When the network moves ahead "1" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    When the network moves ahead "1" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    # last trade mark price = 15900
    # book mark price = 15451
    # median price = 15675
    # => mark price by median =
    When the network moves ahead "1" blocks
    Then the mark price should be "15675" for the market "ETH/FEB23"

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15904 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15675" for the market "ETH/FEB23"
    When the network moves ahead "3" blocks
    # last trade mark price = 15902
    # book mark price = 15451
    # median mark price = (15902+15451)/2 = 15676
    # => mark price by median = 15676
    Then the mark price should be "15676" for the market "ETH/FEB23"

