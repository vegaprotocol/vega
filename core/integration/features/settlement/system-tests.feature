Feature: Test settlement at expiry time from internal oracle

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the average block duration is "1"

    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type         | binding          |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "vegaprotocol.builtin" named "ethDec20Oracle":
      | property                       | type           | binding             | condition                      | value                |
      | vegaprotocol.builtin.timestamp | TYPE_TIMESTAMP | trading termination | OPERATOR_GREATER_THAN_OR_EQUAL | 2019-12-31T23:59:59Z |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding          |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "0" decimal places
    And the settlement data decimals for the oracle named "ethDec21Oracle" is given in "0" decimal places
    
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.02               |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees          | price monitoring   | data source config  |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none  | default-none       | ethDec20Oracle |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1         | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle |

  @STest
  Scenario: Order cannot be placed once the market is expired
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | aux1   | ETH   | 100000 |
      | aux2   | ETH   | 100000 |
      | lpprov | ETH   | 100000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000             | 0.001 | buy  | BID              | 50         | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000             | 0.001 | sell | ASK              | 50         | 1      | submission |

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | aux1   | ETH   | ETH/DEC19 | 264    | 99736   |
      | aux2   | ETH   | ETH/DEC19 | 241    | 99759   |

    # The oracle terminates trading at this time
    # vegaprotocol.builtin.timestamp | TYPE_TIMESTAMP | trading termination | OPERATOR_GREATER_THAN_OR_EQUAL | 2019-12-31T23:59:59Z |
    # So let's make some trades happen Before final settlement

    When time is updated to "2019-12-31T23:59:57Z"
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1005  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | aux1   | ETH/DEC19 | buy  | 1      | 1005  | 1                | TYPE_LIMIT | TIF_GTC | ref-6     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 120    | 9880    |
      | aux1   | ETH   | ETH/DEC19 | 397    | 99601   |
      | aux2   | ETH   | ETH/DEC19 | 241    | 99759   |

    When time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    And time is updated to "2020-01-01T01:01:02Z"
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 10963   |
      | aux1   | ETH   | ETH/DEC19 | 0      | 98077   |
      | aux2   | ETH   | ETH/DEC19 | 0      | 100958  |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     | OrderError: Invalid Market ID |
