Feature: Test one off transfers

Background:
    Given time is updated to "2021-08-26T00:00:00Z"

    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |

    Given the following network parameters are set:
      | name                                    | value |
      | transfer.fee.factor                     |  0.5  |
      | network.markPriceUpdateMaximumFrequency |  0s   |
      | transfer.fee.maxQuantumAmount           |  1    |
      | transfer.feeDiscountDecayFraction       |  0.9  |
      | limits.markets.maxPeggedOrders          |  1500 |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | 1234567890123456789012345678901234567890123456789012345678900001 | BTC   | 1000000000000 |
      | 1234567890123456789012345678901234567890123456789012345678900002 | BTC   | 1000000000000 |
      | aux    | BTC   | 1000000  |
      | aux2   | BTC   | 1000000  |
      | lpprov | BTC   | 90000000 |

    And create the network treasury account for asset "VEGA"

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

Scenario: 
  # party1 needs to place trades so they pay a taker fee
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | 1234567890123456789012345678901234567890123456789012345678900001 | ETH/DEC19 | sell | 100     | 50    | 0                | TYPE_LIMIT  | TIF_GTC |
      | 1234567890123456789012345678901234567890123456789012345678900002 | ETH/DEC19 | buy  | 100     | 0     | 1                | TYPE_MARKET | TIF_FOK |

  # Move forward 3 epochs
  Then the network moves ahead "2" epochs

  # Make sure we know how much BTC party1 has
  Then "1234567890123456789012345678901234567890123456789012345678900001" should have general account balance of "393999999400" for asset "BTC"

  # party1 makes a transfer and should get a reduced transfer fee
  Given the parties submit the following one off transfers:
    | id | from   |  from_account_type    |   to   |   to_account_type    | asset | amount | delivery_time        |
    | 1  | 1234567890123456789012345678901234567890123456789012345678900001 |  ACCOUNT_TYPE_GENERAL | 1234567890123456789012345678901234567890123456789012345678900002 | ACCOUNT_TYPE_GENERAL | BTC   |  10 | 2021-08-26T00:00:00Z |

  # Make sure that party1 has not paid a fee for this transfer (only the transferred amount is removed from the GENERAL account)
  Then "1234567890123456789012345678901234567890123456789012345678900001" should have general account balance of "393999998830" for asset "BTC"
