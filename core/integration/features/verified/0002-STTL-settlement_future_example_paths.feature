Feature: Test settlement future example paths (0002-STTL-011), (0002-STTL-012), (0002-STTL-013)

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the average block duration is "1"

    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "0" decimal places

    And the following network parameters are set:
      | name                                         | value |
      | market.auction.minimumDuration               | 1     |
      | network.markPriceUpdateMaximumFrequency      | 0s    |
      | market.liquidity.successorLaunchWindowLength | 1s    |
      | limits.markets.maxPeggedOrders               | 4     |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none  | default-basic      | ethDec20Oracle     | 0.25                   | 0                         | default-futures |
    And the initial insurance pool balance is "1000" for all the markets

  Scenario: A typical path of a cash settled futures market nearing expiry when market is trading in continuous session (0002-STTL-011)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | aux1   | ETH   | 100000 |
      | aux2   | ETH   | 100000 |
      | lpprov | ETH   | 100000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 10     |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 10     |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000" for the market "ETH/DEC19"

    # check bond
    And the parties should have the following account balances:
      | party | asset | market id   | margin | general | bond  |
      | lpprov  | ETH   | ETH/DEC19 | 6600   | 3400    | 90000 |
    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 564    | 99436   |
      | aux2  | ETH   | ETH/DEC19 | 540    | 99460   |
    # check positions
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | 0            |
      | aux2   | -1     | 0              | 0            |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 564    | 99436   |
      | aux2  | ETH   | ETH/DEC19 | 540    | 99460   |
    # check positions
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | 0            |
      | aux2   | -1     | 0              | 0            |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 0      | 99042   |
      | aux2  | ETH   | ETH/DEC19 | 0      | 100958  |
    
    # check bond
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond   |
      | lpprov  | ETH   | ETH/DEC19 | 0      | 100000  | 0      |
    
    # check positions
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | -958         |
      | aux2   | -1     | 0              | 958          |

    And the network moves ahead "2" blocks
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the global insurance pool balance should be "1000" for the asset "ETH"
  
  Scenario: A less typical path of such a futures market nearing expiry when market is suspended (0002-STTL-012)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 1000   |
      | aux1   | ETH   | 100000 |
      | aux2   | ETH   | 100000 |
      | lpprov | ETH   | 100000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 10     |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 10     |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000" for the market "ETH/DEC19"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-6     |

    And the mark price should be "1000" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 420    | 9580    |
      | party2 | ETH   | ETH/DEC19 | 432    | 468     |

    And then the network moves ahead "10" blocks

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | party2 | ETH/DEC19 | buy  | 1      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"

    # check bond
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond  |
      | lpprov | ETH   | ETH/DEC19 | 0      | 10100   | 90000 |
    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 564    | 99436   |
      | aux2  | ETH   | ETH/DEC19 | 540    | 99460   |
    # check positions
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | 0            |
      | aux2   | -1     | 0              | 0            |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 564    | 99436   |
      | aux2  | ETH   | ETH/DEC19 | 540    | 99460   |
    # check positions
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | 0            |
      | aux2   | -1     | 0              | 0            |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"

    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 0      | 99042   |
      | aux2  | ETH   | ETH/DEC19 | 0      | 100958  |
    
    # check bond
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond   |
      | lpprov  | ETH   | ETH/DEC19 | 0      | 100100  | 0      |
    
    # check positions
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | -958         |
      | aux2   | -1     | 0              | 958          |

    Then the network moves ahead "2" blocks
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the global insurance pool balance should be "942" for the asset "ETH"

    Scenario: Settlement data to cash settled future is submitted before trading is terminated (0002-STTL-013)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | aux1   | ETH   | 1000000 |
      | aux2   | ETH   | 1000000 |
      | lpprov | ETH   | 100000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 10     |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 10     |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000" for the market "ETH/DEC19"
    
    # check bond
    And the parties should have the following account balances:
      | party | asset | market id   | margin | general | bond  |
      | lpprov  | ETH   | ETH/DEC19 | 6600   | 3400    | 90000 |
    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 564    | 999436   |
      | aux2  | ETH   | ETH/DEC19 | 540    | 999460   |
    # check positions
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 1      | 0              | 0            |
      | aux2   | -1     | 0              | 0            |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    Then time is updated to "2020-01-01T01:01:02Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-6     |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | aux1   | 1000  | 1    | aux2   |

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 45    |
    Then time is updated to "2020-01-01T01:01:02Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-8     |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | aux1   | 1000  | 1    | aux2   |
  
    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"
    
    # check margin
    And the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | ETH   | ETH/DEC19 | 0      | 997135  |
      | aux2  | ETH   | ETH/DEC19 | 0      | 1002665 |
    # check positions
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux1   | 3      | 0              | -2865        |
      | aux2   | -3     | 0              | 2865         |

    # check bond
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond   |
      | lpprov  | ETH   | ETH/DEC19 | 0      | 100200  | 0      |
  
    And the network moves ahead "2" blocks
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the global insurance pool balance should be "1000" for the asset "ETH"