Feature: Test settlement at expiry

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"

    And the oracle spec for settlement price filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type           | binding            |
      | prices.ETH.value | TYPE_INTEGER   | settlement price   |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property           | type           | binding              |
      | trading.terminated | TYPE_BOOLEAN   | trading termination  |  

    And the oracle spec for settlement price filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type           | binding            |
      | prices.ETH.value | TYPE_INTEGER   | settlement price   |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type           | binding              |
      | trading.terminated | TYPE_BOOLEAN   | trading termination  |  


    And the oracle spec for settlement price filtering data from "0xCAFECAFE2" named "ethDec22Oracle":
      | property         | type           | binding            |
      | prices.ETH.value | TYPE_INTEGER   | settlement price   |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE2" named "ethDec22Oracle":
      | property           | type           | binding              |
      | trading.terminated | TYPE_BOOLEAN   | trading termination  |  

    And the following network parameters are set:
      | name                                                | value |
      | market.auction.minimumDuration                      | 1     |
    
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 2                  |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | maturity date        | risk model                  | margin calculator         | auction duration | fees          | price monitoring   | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | 2019-12-31T23:59:59Z | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none  | default-none       | default-eth-for-future |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none  | default-none       | ethDec20Oracle         |
      | ETH/DEC21 | ETH        | ETH   | 2021-12-31T23:59:59Z | simple-risk-model-1         | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle         |
      | ETH/DEC22 | ETH        | ETH   | 2022-12-31T23:59:59Z | simple-risk-model-1         | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec22Oracle         |
     
  Scenario: Order cannot be placed once the market is expired
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | aux1   | ETH   | 100000 |
      | aux2   | ETH   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1   | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2   | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1   | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    # Set mark price
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1   | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | aux2   | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-6     |

    When the oracles broadcast data signed with "0xDEADBEEF":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     | OrderError: Invalid Market ID |

  Scenario: Settlement happened when market is being closed - no loss socialisation needed - no insurance taken
    Given the initial insurance pool balance is "10000" for the markets:
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | party3   | ETH   | 5000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |
    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | buy  | BID              | 50         | -10    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | sell | ASK              | 50         | 10     |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    # Set mark price
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-3     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | party2 | ETH   | ETH/DEC19 | 132    | 868     |
      | party3 | ETH   | ETH/DEC19 | 132    | 4868    |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "100256000"

    # Close positions by aux parties
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the oracles broadcast data signed with "0xDEADBEEF":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                         |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     | OrderError: Invalid Market ID |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11676   |
      | party2 | ETH   | ETH/DEC19 | 0      | 42      |
      | party3 | ETH   | ETH/DEC19 | 0      | 4042    |

    # And the cumulated balance for all accounts should be worth "100214513"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "2500" for the asset "ETH"
    And the insurance pool balance should be "12500" for the market "ETH/DEC20"
    And the insurance pool balance should be "12500" for the market "ETH/DEC21"
    And the insurance pool balance should be "12500" for the market "ETH/DEC22"

  Scenario: Settlement happened when market is being closed - no loss socialisation needed - insurance covers losses
    Given the initial insurance pool balance is "10000" for the markets:
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |
    And the parties submit the following liquidity provision:
      | id  | party     | market id | commitment amount | fee | side | pegged reference | proportion | offset |
      | lp1 | party-lp | ETH/DEC19 | 30000000           | 0   | buy  | BID              | 50         | -10    |
      | lp1 | party-lp | ETH/DEC19 | 30000000           | 0   | sell | ASK              | 50         |  10    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    # Set mark price
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1   | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2   | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 2      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    
    And debug transfers
    
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | party2 | ETH   | ETH/DEC19 | 264    | 736     |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the cumulated balance for all accounts should be worth "100251000"

    # Close positions by aux parties
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the oracles broadcast data signed with "0xDEADBEEF":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    Then the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11676   |
      | party2 | ETH   | ETH/DEC19 | 0      | 0       |

    # And the cumulated balance for all accounts should be worth "100214513"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    # 916 were taken from the insurance pool to cover the losses of party 2, the remaining is split between global and the other market
    And the insurance pool balance should be "2271" for the asset "ETH"
    And the insurance pool balance should be "12271" for the market "ETH/DEC20"
    And the insurance pool balance should be "12271" for the market "ETH/DEC21"
    And the insurance pool balance should be "12271" for the market "ETH/DEC22"

  Scenario: Settlement happened when market is being closed - loss socialisation in action - insurance doesn't covers all losses
     Given the initial insurance pool balance is "500" for the markets:
     Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |
    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | buy  | BID              | 50         | -10    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0   | sell | ASK              | 50         | 10     |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    # Set mark price
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
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
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    When the oracles broadcast data signed with "0xDEADBEEF":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And time is updated to "2020-01-01T01:01:02Z"

    And debug transfers
    
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 11399   |
      | party2 | ETH   | ETH/DEC19 | 0      | 0       |
    # And the cumulated balance for all accounts should be worth "100214513"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    # 500 were taken from the insurance pool to cover the losses of party 2, still not enough to cover losses of (1000-42)*2 for party2
    And the insurance pool balance should be "0" for the asset "ETH"
    And the insurance pool balance should be "500" for the market "ETH/DEC20"
    And the insurance pool balance should be "500" for the market "ETH/DEC21"
    And the insurance pool balance should be "500" for the market "ETH/DEC22"
    
  Scenario: Settlement happened when market is being closed after being in Suspended status and in a protective auction - no loss socialisation needed - no insurance taken

    Given the initial insurance pool balance is "10000" for the markets:
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader4  | ETH   | 10000     |
      | party-lp | ETH   | 100000000 |

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
 
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    And the mark price should be "1002" for the market "ETH/DEC21"
  
    Then the parties place the following orders:
      | party    | market id  | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | sell | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | buy  | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer   | price | size | seller   |
      | trader4 | 900   | 2    | trader3a |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2856   | 5343    |
      | trader4  | ETH   | ETH/DEC21 | 1392   | 6807    |

    And the settlement account should have a balance of "0" for the market "ETH/DEC21"
    And the cumulated balance for all accounts should be worth "300060000"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2022-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2022-01-01T01:01:02Z"
    
    And debug transfers

    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 7059    |
      | trader4  | ETH   | ETH/DEC21 | 0      | 6483    |

    And the cumulated balance for all accounts should be worth "300055604"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the insurance pool balance should be "2500" for the asset "ETH"
    And the insurance pool balance should be "12500" for the market "ETH/DEC19"
    And the insurance pool balance should be "12500" for the market "ETH/DEC20"
    And the insurance pool balance should be "12500" for the market "ETH/DEC22"

  Scenario: Settlement happened when market is being closed after being in Suspended status and in a protective auction - no loss socialisation needed - insurance covers losses

   Given the initial insurance pool balance is "10000" for the markets:
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader4  | ETH   |  2400     |
      | party-lp | ETH   | 100000000 |

    Then the parties place the following orders:
      | party    | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
 
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    And the mark price should be "1002" for the market "ETH/DEC21"
  
    Then the parties place the following orders:
      | party    | market id  | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | sell | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | buy  | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer   | price | size | seller   |
      | trader4 | 900   | 2    | trader3a |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2856   | 5343    |
      | trader4  | ETH   | ETH/DEC21 |  599   |    0    |

    And the settlement account should have a balance of "0" for the market "ETH/DEC21"
    And the cumulated balance for all accounts should be worth "300052400"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2022-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2022-01-01T01:01:02Z"

    And debug transfers
    
    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 7059    |
      | trader4  | ETH   | ETH/DEC21 | 0      |    0    |

    And the cumulated balance for all accounts should be worth "300048004"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    # 1117 were taken from the insurance pool to cover the losses of trader4, the remaining is split between global and the other market
    And the insurance pool balance should be "2223" for the asset "ETH"
    And the insurance pool balance should be "12220" for the market "ETH/DEC19"
    And the insurance pool balance should be "12220" for the market "ETH/DEC20"
    And the insurance pool balance should be "12220" for the market "ETH/DEC22"
  
  Scenario: Settlement happened when market is being closed after being in Suspended status and in a protective auction - loss socialisation in action - insurance doesn't covers all losses

   Given the initial insurance pool balance is "1000" for the markets:
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader4  | ETH   |  2400     |
      | party-lp | ETH   | 100000000 |

    Then the parties place the following orders:
      | party    | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1     | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
 
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    And the mark price should be "1002" for the market "ETH/DEC21"
  
    Then the parties place the following orders:
      | party    | market id  | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | sell | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | buy  | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer   | price | size | seller   |
      | trader4 | 900   | 2    | trader3a |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2856   | 5343    |
      | trader4  | ETH   | ETH/DEC21 |  599   |    0    |

    And the settlement account should have a balance of "0" for the market "ETH/DEC21"
    And the cumulated balance for all accounts should be worth "300016400"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2022-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2022-01-01T01:01:02Z"

    And debug transfers
    
    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 6982    |
      | trader4  | ETH   | ETH/DEC21 | 0      |    0    |

    And the cumulated balance for all accounts should be worth "300012004"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    # 500 were taken from the insurance pool to cover the losses of trader4, still not enough to cover losses of for trader4
    And the insurance pool balance should be "0" for the asset "ETH"
    And the insurance pool balance should be "1000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1000" for the market "ETH/DEC22"


  # Scenario: This mechanism does not incur fees to traders that have open positions that are settled at expiry - Add a step for this ?

