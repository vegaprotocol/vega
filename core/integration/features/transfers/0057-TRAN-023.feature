Feature: Test one off transfers

Background:
    Given time is updated to "2021-08-26T00:00:00Z"

    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | BTC/DEC23 | BTC        | BTC   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |

    Given the following network parameters are set:
      | name                                    | value |
      | transfer.fee.factor                     |  0.5  |
      | network.markPriceUpdateMaximumFrequency |  0s   |
      | transfer.fee.maxQuantumAmount           |  1    |
      | transfer.feeDiscountDecayFraction       |  0.9  |
      | limits.markets.maxPeggedOrders          |  1500 |

    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount        |
      | 1234567890123456789012345678901234567890123456789012345678900001 | BTC   | 1000000000000 |
      | 1234567890123456789012345678901234567890123456789012345678900002 | BTC   | 1000000000000 |
      | aux                                                              | BTC   | 1000000       |
      | aux2                                                             | BTC   | 1000000       |
      | lpprov                                                           | BTC   | 90000000      |

    And create the network treasury account for asset "VEGA"

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/DEC23 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | BTC/DEC23 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | BTC/DEC23 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | BTC/DEC23 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | BTC/DEC23 | buy  | 1      | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/DEC23 | sell | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/DEC23 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | BTC/DEC23 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/DEC23"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/DEC23"

Scenario: Make sure we are not charged for a transfer if we have a discount (0057-TRAN-023)
  # party1 needs to place trades so they pay a taker fee
    When the parties place the following orders:
      | party                                                            | market id | side | volume | price | resulting trades | type        | tif     |
      | 1234567890123456789012345678901234567890123456789012345678900002 | BTC/DEC23 | sell | 100    | 50    | 0                | TYPE_LIMIT  | TIF_GTC |
      | 1234567890123456789012345678901234567890123456789012345678900001 | BTC/DEC23 | buy  | 100    | 0     | 1                | TYPE_MARKET | TIF_FOK |

  # Move forward 3 epochs and check the discount after each step
  Then the network moves ahead "1" epochs

  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 500                |

  Then the network moves ahead "1" epochs

  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 450                |

  Then the network moves ahead "1" epochs

  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 405                |

  # Make sure we know how much BTC party1 has
  Then "1234567890123456789012345678901234567890123456789012345678900001" should have general account balance of "393999998840" for asset "BTC"

  # party1 makes a transfer and should get a reduced transfer fee
  Given the parties submit the following one off transfers:
    | id | from                                                             |  from_account_type   | to                                                               |   to_account_type    | asset | amount | delivery_time        |
    | 1  | 1234567890123456789012345678901234567890123456789012345678900001 | ACCOUNT_TYPE_GENERAL | 1234567890123456789012345678901234567890123456789012345678900002 | ACCOUNT_TYPE_GENERAL | BTC   | 100    | 2021-08-26T00:00:00Z |

  # Make sure that party1 has not paid a fee for this transfer (only the transferred amount is removed from the GENERAL account) 393999998840-100
  Then "1234567890123456789012345678901234567890123456789012345678900001" should have general account balance of "393999998740" for asset "BTC"

  # After the transfer the discount amount will be reduced by the amount we didn't get charged (50)
  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 355                |



Scenario: Make sure we are only charged for a transfer after the discount is used up (0057-TRAN-023)
  # party1 needs to place trades so they pay a taker fee
    When the parties place the following orders:
      | party                                                            | market id | side | volume | price | resulting trades | type        | tif     |
      | 1234567890123456789012345678901234567890123456789012345678900002 | BTC/DEC23 | sell | 100    | 50    | 0                | TYPE_LIMIT  | TIF_GTC |
      | 1234567890123456789012345678901234567890123456789012345678900001 | BTC/DEC23 | buy  | 100    | 0     | 1                | TYPE_MARKET | TIF_FOK |

  # Move forward 3 epochs and check the discount after each step
  Then the network moves ahead "1" epochs

  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 500                |

  Then the network moves ahead "1" epochs

  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 450                |

  Then the network moves ahead "1" epochs

  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 405                |

  # Make sure we know how much BTC party1 has
  Then "1234567890123456789012345678901234567890123456789012345678900001" should have general account balance of "393999998840" for asset "BTC"

  # party1 makes a transfer and should get a reduced transfer fee
  Given the parties submit the following one off transfers:
    | id | from                                                             |  from_account_type   | to                                                               |   to_account_type    | asset | amount | delivery_time        |
    | 1  | 1234567890123456789012345678901234567890123456789012345678900001 | ACCOUNT_TYPE_GENERAL | 1234567890123456789012345678901234567890123456789012345678900002 | ACCOUNT_TYPE_GENERAL | BTC   | 1000   | 2021-08-26T00:00:00Z |

  # fee = 1000 * 0.5 == 500. fee after discount = 500-405 = 95
  # 393999998840-(1000+95) 
  Then "1234567890123456789012345678901234567890123456789012345678900001" should have general account balance of "393999997745" for asset "BTC"

  # After the transfer the discount amount will be reduced by the amount we didn't get charged (50)
  And the parties have the following transfer fee discounts:
  | party                                                              | asset | available discount |
  | 1234567890123456789012345678901234567890123456789012345678900001   | BTC   | 0                  |
