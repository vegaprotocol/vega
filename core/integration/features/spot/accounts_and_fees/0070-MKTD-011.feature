Feature: Spot market with decimal places

  Scenario: 0070-MKTD-011 As a user all transfers (margin top-up, release, MTM settlement) are calculated and communicated (via events) in asset precision

  Background:

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.value.windowLength               | 1h    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 3              |
      | BTC | 3              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 3              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100000 |
      | party2 | BTC   | 1000   |
    And the average block duration is "1"

    # At this point we have a total of 10,000 ETH in the system and 100 BTC. These are all stored in the user accounts.

    # Place some orders to allow us out of the opening auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    And clear transfer response events
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # After the auction there were 2 trades so the assets have moved about.
    # There are no fees in the opening auction so all the assets are stored in the users accounts
    # ETH = 9,990 + 10 = 10,000
    # BTC = 99 + 1 = 100

    Then "party1" should have general account balance of "99990" for asset "ETH"
    Then "party1" should have holding account balance of "0" for asset "ETH"
    Then "party1" should have general account balance of "10" for asset "BTC"

    Then "party2" should have general account balance of "10" for asset "ETH"
    Then "party2" should have general account balance of "990" for asset "BTC"
    Then "party2" should have holding account balance of "0" for asset "BTC"

    # No fees in opening auction
    And the accumulated liquidity fees should be "0" for the market "BTC/ETH"
    And the accumulated infrastructure fees should be "0" for the asset "BTC"
    And the accumulated infrastructure fees should be "0" for the asset "ETH"

    # Place some orders to create some trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1008  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1008  | 1                | TYPE_LIMIT | TIF_GTC |

    # These orders have created trades and paid fees on them.
    Then the following transfers should happen:
      | from   | to     | from account            | to account                       | market id | amount | asset |
      | party1 | party1 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_HOLDING             |           | 10     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 1      | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 1      | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | 0      | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 1      | ETH   |
      | party1 | party1 | ACCOUNT_TYPE_HOLDING    | ACCOUNT_TYPE_GENERAL             |           | 10     | ETH   |
      | party2 | party1 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_GENERAL             |           | 10     | BTC   |
      | party1 | party2 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_GENERAL             |           | 10     | ETH   |

    # party1 started with 99990 ETH but sold 9 to leave 99981
    #        started with 1 BTC but bought 1 to give 2
    # party2 started with 10 ETH but bought 8 to give 18
    #        started with 99 BTC but sold one to give 98
    # infra  started with 0 ETH but gained 1 due to fees

    # ETH = 99,981 + 18 + 1 = 10,000
    # BTC = 20 + 980 = 1000
    Then "party1" should have general account balance of "99981" for asset "ETH"
    Then "party1" should have holding account balance of "0" for asset "ETH"
    Then "party1" should have general account balance of "20" for asset "BTC"

    Then "party2" should have general account balance of "18" for asset "ETH"
    Then "party2" should have general account balance of "980" for asset "BTC"
    Then "party2" should have holding account balance of "0" for asset "BTC"

    And the accumulated liquidity fees should be "0" for the market "BTC/ETH"
    And the accumulated infrastructure fees should be "0" for the asset "BTC"
    And the accumulated infrastructure fees should be "1" for the asset "ETH"
