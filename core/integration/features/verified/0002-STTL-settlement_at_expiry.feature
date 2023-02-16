Feature: Test settlement at expiry (0016-PFUT-012)

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the average block duration is "1"

    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
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
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none  | default-none       | ethDec20Oracle     | 1e6                    | 1e6                       |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1         | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 1e6                    | 1e6                       |

  Scenario: Order cannot be placed once the market is expired (0002-STTL-001)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | aux1   | ETH   | 100000 |
      | aux2   | ETH   | 100000 |
      | lpprov | ETH   | 100000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     | OrderError: Invalid Market ID |

  Scenario: Settlement happened when market is being closed - no loss socialisation needed - no insurance taken (0002-STTL-002, 0002-STTL-007, 0005-COLL-002, 0015-INSR-002)
    Given the initial insurance pool balance is "10000" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | party3   | ETH   | 5000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |

    And the cumulated balance for all accounts should be worth "100236000"

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | sell | ASK              | 50         | 10     | amendment  |
      | lp2 | party-lp | ETH/DEC21 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp2 | party-lp | ETH/DEC21 | 30000000          | 0   | sell | ASK              | 50         | 10     | amendment  |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 2      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    # Other market
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC21 | sell | 2      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    Then the opening auction period ends for market "ETH/DEC19"
    Then the opening auction period ends for market "ETH/DEC21"

    And the mark price should be "1000" for the market "ETH/DEC19"

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    Then the network moves ahead "2" blocks

    # The market considered here ("ETH/DEC19") relies on "0xCAFECAFE" oracle, checking that broadcasting events from "0xCAFECAFE1" should have no effect on it apart from insurance pool transfer
    And the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And the network moves ahead "2" blocks

    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 2000  |

    And the network moves ahead "2" blocks

    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-3     |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 1000  | 1    | party1 |
      | party3 | 1000  | 1    | party1 |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | party2 | ETH   | ETH/DEC19 | 132    | 868     |
      | party3 | ETH   | ETH/DEC19 | 132    | 4868    |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "100236000"

    # Close positions by aux parties
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |


    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | 0              | 0            |
      | party2 | 1      | 0              | 0            |
      | party3 | 1      | 0              | 0            |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |

    # Order can't be placed after oracle data is received (expecting party positions to remain unchanged)
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error               |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC | trading not allowed |

    And time is updated to "2020-01-01T01:01:01Z"

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | 0              | 0            |
      | party2 | 1      | 0              | 0            |
      | party3 | 1      | 0              | 0            |

    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |

    Then time is updated to "2020-01-01T01:01:02Z"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     | OrderError: Invalid Market ID |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11916   |
      | party2 | ETH   | ETH/DEC19 | 0      | 42      |
      | party3 | ETH   | ETH/DEC19 | 0      | 4042    |

    And the cumulated balance for all accounts should be worth "100236000"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the network treasury balance should be "20000" for the asset "ETH"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"

  Scenario: Same as above, but the other market already terminated before the end of scenario, expecting 0 balances in per market insurance pools - all should go to per asset insurance pool (0002-STTL-additional-tests, 0005-COLL-002, 0015-INSR-002)

    Given the initial insurance pool balance is "10000" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | party3   | ETH   | 5000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |
      | lpprov   | ETH   | 100000000 |

    And the cumulated balance for all accounts should be worth "200236000"

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | sell | ASK              | 50         | 10     | amendment  |
      | lp2 | lpprov   | ETH/DEC21 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp2 | lpprov   | ETH/DEC21 | 30000000          | 0   | sell | ASK              | 50         | 10     | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    # Other market
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC21 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    Then the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC21 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-3     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    Then the network moves ahead "2" blocks

    # The market considered here ("ETH/DEC19") relies on "0xCAFECAFE" oracle, checking that broadcasting events from "0xCAFECAFE1" should have no effect on it apart from insurance pool transfer
    And the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And the network moves ahead "2" blocks

    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    And the insurance pool balance should be "10000" for the market "ETH/DEC21"
    And the insurance pool balance should be "10000" for the market "ETH/DEC19"
    And the network treasury balance should be "0" for the asset "ETH"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 700   |

    And the network moves ahead "1" blocks
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the insurance pool balance should be "15000" for the market "ETH/DEC19"
    And the network treasury balance should be "5000" for the asset "ETH"

    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-3     |

    And the cumulated balance for all accounts should be worth "200236000"

    # Close positions by aux parties
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2020-01-01T01:01:01Z"

    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |

    Then time is updated to "2020-01-01T01:01:02Z"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     | OrderError: Invalid Market ID |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11916   |
      | party2 | ETH   | ETH/DEC19 | 0      | 42      |
      | party3 | ETH   | ETH/DEC19 | 0      | 4042    |

    And the cumulated balance for all accounts should be worth "200236000"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the network treasury balance should be "20000" for the asset "ETH"

  Scenario: Settlement happened when market is being closed - no loss socialisation needed - insurance covers losses (0002-STTL-008)
    Given the initial insurance pool balance is "1000" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |
    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | sell | ASK              | 50         | 10     | amendment  |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | party2 | ETH   | ETH/DEC19 | 264    | 736     |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "100213000"

    # Close positions by aux parties
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11916   |
      | party2 | ETH   | ETH/DEC19 | 0      | 0       |

    And the cumulated balance for all accounts should be worth "100213000"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    # 916 were taken from the insurance pool to cover the losses of party 2, the remaining is split between global and the other market
    And the network treasury balance should be "42" for the asset "ETH"
    And the insurance pool balance should be "1042" for the market "ETH/DEC21"

  Scenario: Settlement happened when market is being closed - loss socialisation in action - insurance doesn't cover all losses (0002-STTL-009)
    Given the initial insurance pool balance is "500" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | aux1     | ETH   | 1000000   |
      | aux2     | ETH   | 1000000   |
      | party-lp | ETH   | 100000000 |
    And the cumulated balance for all accounts should be worth "102012000"

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | sell | ASK              | 50         | 10     | amendment  |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | party2 | ETH   | ETH/DEC19 | 264    | 736     |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "102012000"

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    And time is updated to "2020-01-01T01:01:02Z"


    # 416 missing, but party1 & aux1 get a haircut of 209 each due to flooring
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11709   |
      | party2 | ETH   | ETH/DEC19 | 0      | 0       |
    And the cumulated balance for all accounts should be worth "102012000"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    # 500 were taken from the insurance pool to cover the losses of party 2, still not enough to cover losses of (1000-42)*2 for party2
    And the network treasury balance should be "0" for the asset "ETH"
    And the insurance pool balance should be "500" for the market "ETH/DEC21"

  Scenario: Settlement happened when market is being closed whilst being suspended (due to protective auction) - loss socialisation in action - insurance doesn't covers all losses (0002-STTL-004, 0002-STTL-009)

    Given the initial insurance pool balance is "500" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | aux1     | ETH   | 1000000   |
      | aux2     | ETH   | 1000000   |
      | party-lp | ETH   | 100000000 |
    And the cumulated balance for all accounts should be worth "102012000"

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC21 | 30000000          | 0   | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC21 | 30000000          | 0   | sell | ASK              | 50         | 10     | amendment  |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 1      | 890   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC21 | sell | 1      | 1110  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC21 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC21 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | party2 | ETH/DEC21 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-6     |

    And the mark price should be "1000" for the market "ETH/DEC21"

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC21 | 252    | 9753    |
      | party2 | ETH   | ETH/DEC21 | 372    | 603     |

    And then the network moves ahead "10" blocks

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 1      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | party2 | ETH/DEC21 | buy  | 1      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC21"
    And the market state should be "STATE_SUSPENDED" for the market "ETH/DEC21"

    And then the network moves ahead "10" blocks

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And then the network moves ahead "1" blocks

    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    And then the network moves ahead "400" blocks

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |

    And then the network moves ahead "10" blocks

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 800   |

    And then the network moves ahead "10" blocks

    # Check that party positions and overall account balances are the same as before auction start (accounting for a settlement transfer of 200 from party2 to party1)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC21 | 0      | 10205   |
      | party2 | ETH   | ETH/DEC21 | 0      | 775     |

    And the cumulated balance for all accounts should be worth "102012000"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the network treasury balance should be "250" for the asset "ETH"
    And the insurance pool balance should be "750" for the market "ETH/DEC19"
