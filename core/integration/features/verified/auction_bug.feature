Feature: Indicative price within bounds but mark price outside bounds

  Scenario:


    Given the average block duration is "1"

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    And the following assets are registered:
      | id       | decimal places | quantum |
      | USDT.0.1 | 0              | 1       |

    Given the price monitoring named "pm":
      | horizon | probability | auction extension |
      | 60      | 0.999999999 | 5                 |
      | 60      | 0.999999999 | 5                 |
      | 120     | 0.999999999 | 10                |
      | 120     | 0.999999999 | 10                |



    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property  | price type   | price decimals |
      | oracle1 | price.USD.value | TYPE_INTEGER | 0              |
    And the markets:
      | id       | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 |
      | ETH/USDT | USDT       | USDT.0.1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | pm               | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       | weight     | 1            | 1           | 0           | 0,0,1,0        | 1m0s,1m0s,1m0s,1m0s        | oracle1 |


    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount   |
      | aux1   | USDT.0.1 | 10000000 |
      | aux2   | USDT.0.1 | 10000000 |
      | party1 | USDT.0.1 | 10000000 |
      | party2 | USDT.0.1 | 10000000 |

    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDT  | buy  | 100    | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 100    | 1001  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/USDT"
    And the market data for the market "ETH/USDT" should be:
      | mark price | trading mode                    | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | 60      | 984       | 1016      |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 977       | 1024      |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 977       | 1024      |

    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name            | value | time offset |
      | price.USD.value | 800   | 0s          |


    Given the network moves ahead "1" blocks
    # And the parties place the following orders:
    #   | party  | market id | side | volume | price | resulting trades | type       | tif     |
    #   | party1 | ETH/USDT  | buy  | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/USDT" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | indicative price | timestamp           | auction start       | auction end         |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 984       | 1016      | 0                | 1575072003000000000 | 1575072002000000000 | 1575072007000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 977       | 1024      | 0                | 1575072003000000000 | 1575072002000000000 | 1575072007000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 977       | 1024      | 0                | 1575072003000000000 | 1575072002000000000 | 1575072007000000000 |

    # Advance 5 seconds to end of auction
    Given the network moves ahead "5" blocks
    # Auction ends at indicative price, trades excecuted
    # And the following trades should be executed:
    #   | buyer  | price | size | seller |
    #   | party1 | 1001  | 1    | aux2   |
    # Market instantly reenters auction as latest price from oracle is outside bounds
    And the market data for the market "ETH/USDT" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | indicative price | timestamp           | auction start       | auction end         |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 984       | 1016      | 0                | 1575072008000000000 | 1575072008000000000 | 1575072013000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 977       | 1024      | 0                | 1575072008000000000 | 1575072008000000000 | 1575072013000000000 |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 120     | 977       | 1024      | 0                | 1575072008000000000 | 1575072008000000000 | 1575072013000000000 |





