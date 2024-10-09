Feature: Distributing rewards to parties based on trading activity in Spot markets

  Parties actively trading on Spot markets should be eligible to recieve
  rewards based on the maker fees they pay or receive as well as any
  liquidity fees received.

  Tests validate trading activity contributes correctly to each metric
  on Spot markets.

  Background:

    # Initialise the network
    Given the following network parameters are set:
      | name                        | value |
      | validators.epoch.length     | 60s   |
      | market.fee.factors.makerFee | 0.01  |
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USDT.0.1 | 0              | 1       |
      | BTC.0.1  | 0              | 1       |
      | ETH.0.1  | 0              | 1       |
    And the average block duration is "1"

    # Setup the parties
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset    | amount      |
      | lp1                                                              | BTC.0.1  | 10000000    |
      | lp2                                                              | BTC.0.1  | 10000000    |
      | aux1                                                             | BTC.0.1  | 10000000    |
      | aux2                                                             | BTC.0.1  | 10000000    |
      | buyer1                                                           | BTC.0.1  | 10000000    |
      | buyer2                                                           | BTC.0.1  | 10000000    |
      | seller1                                                          | BTC.0.1  | 10000000    |
      | seller2                                                          | BTC.0.1  | 10000000    |
      | lp1                                                              | ETH.0.1  | 10000000    |
      | lp2                                                              | ETH.0.1  | 10000000    |
      | aux1                                                             | ETH.0.1  | 10000000    |
      | aux2                                                             | ETH.0.1  | 10000000    |
      | buyer1                                                           | ETH.0.1  | 10000000    |
      | buyer2                                                           | ETH.0.1  | 10000000    |
      | seller1                                                          | ETH.0.1  | 10000000    |
      | seller2                                                          | ETH.0.1  | 10000000    |
      | lp1                                                              | USDT.0.1 | 10000000000 |
      | lp2                                                              | USDT.0.1 | 10000000000 |
      | aux1                                                             | USDT.0.1 | 10000000000 |
      | aux2                                                             | USDT.0.1 | 10000000000 |
      | buyer1                                                           | USDT.0.1 | 10000000000 |
      | buyer2                                                           | USDT.0.1 | 10000000000 |
      | seller1                                                          | USDT.0.1 | 10000000000 |
      | seller2                                                          | USDT.0.1 | 10000000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USDT.0.1 | 10000000000 |

    # Setup the BTC/USDT (zero dp) and ETH/USDT (non-zero dp) spot markets and the GOLD/USDT leveraged market (future or perpetual depending on test)
    Given the spot markets:
      | id       | name     | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places | position decimal places | sla params    |
      | BTC/USDT | BTC/USDT | BTC.0.1    | USDT.0.1    | default-log-normal-risk-model | 1                | default-none | default-none     | 0              | 0                       | default-basic |
      | ETH/USDT | ETH/USDT | ETH.0.1    | USDT.0.1    | default-log-normal-risk-model | 1                | default-none | default-none     | 1              | 1                       | default-basic |
    And the markets:
      | id        | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | GOLD/USDT | USDT       | USDT.0.1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    And the parties submit the following liquidity provision:
      | id       | party | market id | commitment amount | fee  | lp type    |
      | lp1-BTC  | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp1-ETH  | lp1   | ETH/USDT  | 10000             | 0.01 | submission |
      | lp2-BTC  | lp2   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2-ETH  | lp2   | ETH/USDT  | 10000             | 0.01 | submission |
      | lp1-GOLD | lp1   | GOLD/USDT | 10000             | 0.01 | submission |
      | lp2-GOLD | lp2   | GOLD/USDT | 10000             | 0.01 | submission |
    # On the BTC/USDT market, only lp1 will meet their commitment and receive liquidity rewards
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1   | BTC/USDT  | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1   | BTC/USDT  | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    # On the ETH/USDT market, only lp1 will meet their commitment and receive liquidity rewards
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp1   | ETH/USDT  | buy  | 100    | 9990  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDT  | buy  | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1   | ETH/USDT  | sell | 100    | 10010 | 0                | TYPE_LIMIT | TIF_GTC |
    # On the GOLD/USDT market, only lp2 will meet their commitment and receive liquidity rewards
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp2   | GOLD/USDT | buy  | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | GOLD/USDT | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | GOLD/USDT | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | GOLD/USDT | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "GOLD/USDT"


  Scenario: In multiple spot markets, buyers pay maker fees and earn maker fees paid rewards. (0056-REWA-152)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller maker fee |
      | buyer1 | seller1 | 1000  | 10   | 100             | 0                |
      | buyer2 | seller2 | 10000 | 100  | 100             | 0                |

    # Move to the end of the epoch - buyer1 and buyer2 receive equal
    # rewards as they paid the same maker fees on their respective
    # spot market.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 5000    |
      | buyer2  | USDT.0.1 | 5000    |
      | seller1 | USDT.0.1 | 0       |
      | seller2 | USDT.0.1 | 0       |


  Scenario: In multiple spot market, sellers pay maker fees and earn maker fees paid rewards. (0056-REWA-153)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller maker fee |
      | buyer1 | seller1 | 1000  | 10   | 0               | 100              |
      | buyer2 | seller2 | 10000 | 100  | 0               | 100              |

    # Move to the end of the epoch - seller1 and seller2 receive equal
    # rewards as they paid the same maker fees on their respective
    # spot market.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 0       |
      | buyer1  | USDT.0.1 | 0       |
      | seller1 | USDT.0.1 | 5000    |
      | seller1 | USDT.0.1 | 5000    |


  Scenario: In multiple spot markets, buyers receive maker fees and earn maker fees received rewards. (0056-REWA-154)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                         | asset    | amount | start_epoch | end_epoch | factor | metric                              | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller maker fee |
      | buyer1 | seller1 | 1000  | 10   | 0               | 100              |
      | buyer2 | seller2 | 10000 | 100  | 0               | 100              |

    # Move to the end of the epoch - buyer1 and buyer2 receive equal
    # rewards as they received the same maker fees on their respective
    # spot market.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 5000    |
      | buyer2  | USDT.0.1 | 5000    |
      | seller1 | USDT.0.1 | 0       |
      | seller2 | USDT.0.1 | 0       |


  Scenario: In multiple spot markets, sellers receive maker fees and earn maker fees received rewards. (0056-REWA-155)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                         | asset    | amount | start_epoch | end_epoch | factor | metric                              | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller maker fee |
      | buyer1 | seller1 | 1000  | 10   | 100             | 0                |
      | buyer2 | seller2 | 10000 | 100  | 100             | 0                |

    # Move to the end of the epoch - seller1 and seller2 receive equal
    # rewards as they received the same maker fees on their respective
    # spot market.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 0       |
      | buyer2  | USDT.0.1 | 0       |
      | seller1 | USDT.0.1 | 5000    |
      | seller2 | USDT.0.1 | 5000    |


  Scenario: In multiple spot markets, liquidity providers who receive liquidity fees, earn liquidity fees received rewards. (0056-REWA-156)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer liquidity fee | seller liquidity fee |
      | buyer1 | seller1 | 1000  | 10   | 100                 | 0                    |
      | buyer2 | seller2 | 10000 | 100  | 100                 | 0                    |

    # Move to the end of the epoch - lp1 receives all the rewards as
    # they were the only lp to meet their commitment.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | lp1     | USDT.0.1 | 10000   |
      | lp2     | USDT.0.1 | 0       |
      | buyer1  | USDT.0.1 | 0       |
      | buyer2  | USDT.0.1 | 0       |
      | seller1 | USDT.0.1 | 0       |
      | seller2 | USDT.0.1 | 0       |


  Scenario Outline: Given spot markets where the base asset matches the dispatch metric of a reward, rewards will not be paid out as fees are paid in the quote asset. (0056-REWA-157)(0056-REWA-158)(0056-REWA-159)(0056-REWA-160)(0056-REWA-161)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id         | from                                                             | from_account_type    | to                                                               | to_account_type | asset    | amount | start_epoch | end_epoch | factor | metric            | metric_asset | markets | lock_period |
      | reward-btc | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | <account type>  | USDT.0.1 | 10000  | 1           |           | 1      | <dispatch metric> | BTC.0.1      |         | 100         |
      | reward-eth | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | <account type>  | USDT.0.1 | 10000  | 1           |           | 1      | <dispatch metric> | ETH.0.1      |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller2 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller1 | ETH/USDT  | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer1  | ETH/USDT  | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller maker fee | buyer liquidity fee | seller liquidity fee |
      | buyer1 | seller1 | 1000  | 10   | 100             | 0                | 100                 | 0                    |
      | buyer2 | seller2 | 1000  | 10   | 0               | 100              | 0                   | 100                  |
      | buyer1 | seller1 | 10000 | 100  | 100             | 0                | 100                 | 0                    |
      | buyer2 | seller2 | 10000 | 100  | 0               | 100              | 0                   | 100                  |

    # Move to the end of the epoch - no parties should receive any
    # rewards as the asset for metric scoped the base asset.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | lp1     | USDT.0.1 | 0       |
      | buyer1  | USDT.0.1 | 0       |
      | buyer2  | USDT.0.1 | 0       |
      | seller1 | USDT.0.1 | 0       |
      | seller2 | USDT.0.1 | 0       |

    Examples:
      | account type                            | dispatch metric                     |
      | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | DISPATCH_METRIC_MAKER_FEES_PAID     |
      | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | DISPATCH_METRIC_MAKER_FEES_RECEIVED |
      | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | DISPATCH_METRIC_LP_FEES_RECEIVED    |


  Scenario Outline: Given spot markets, trading activity does not contribute to metrics which can not be evaluated. (0056-REWA-162)(0056-REWA-163)(0056-REWA-164)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type | asset    | amount | start_epoch | end_epoch | factor | metric            | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | <account type>  | USDT.0.1 | 10000  | 1           |           | 1      | <dispatch metric> | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on two different markets using a different number of decimal places
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | ETH/USDT  | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | ETH/USDT  | buy  | 100    | 10000 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller maker fee |
      | buyer1 | seller1 | 1000  | 10   | 100             | 0                |
      | buyer2 | seller2 | 10000 | 100  | 100             | 0                |

    # Move to the end of the epoch - no parties should receive any
    # rewards as the dispatch metric can not be evaluated on a Spot
    # market.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 0       |
      | buyer2  | USDT.0.1 | 0       |
      | seller1 | USDT.0.1 | 0       |
      | seller2 | USDT.0.1 | 0       |

    Examples:
      | account type                         | dispatch metric                  |
      | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | DISPATCH_METRIC_AVERAGE_NOTIONAL |
      | ACCOUNT_TYPE_REWARD_RELATIVE_RETURN  | DISPATCH_METRIC_RELATIVE_RETURN  |
      | ACCOUNT_TYPE_REWARD_REALISED_RETURN  | DISPATCH_METRIC_REALISED_RETURN  |


  Scenario: Given a maker fees paid reward, contributions from a spot market are correctly aggregated with markets allowing leverage. (0056-REWA-165)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on the spot market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller liquidity fee |
      | buyer1 | seller1 | 1000  | 10   | 0               | 100                  |

    Given clear trade events

    # Generate trades on the leveraged market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | GOLD/USDT | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | GOLD/USDT | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer liquidity fee | seller maker fee |
      | buyer2 | seller2 | 1000  | 10   | 100                 | 0                |

    # Move to the end of the epoch - seller1 and buyer2 should receive
    # equal rewards as they PAID an equal amount of maker fees on the
    # spot and leveraged markets respectively.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 0       |
      | seller1 | USDT.0.1 | 5000    |
      | buyer2  | USDT.0.1 | 5000    |
      | seller2 | USDT.0.1 | 0       |


  Scenario: Given a maker fees received reward, contributions from a spot market are correctly aggregated with markets allowing leverage. (0056-REWA-166)

    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                         | asset    | amount | start_epoch | end_epoch | factor | metric                              | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on the spot market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller liquidity fee |
      | buyer1 | seller1 | 1000  | 10   | 0               | 100                  |

    Given clear trade events

    # Generate trades on the leveraged market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | GOLD/USDT | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | GOLD/USDT | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer liquidity fee | seller maker fee |
      | buyer2 | seller2 | 1000  | 10   | 100                 | 0                |

    # Move to the end of the epoch - buyer1 and seller2 should receive
    # equal rewards as they received an equal amount of maker fees on
    # the spot and leveraged markets respectively.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party   | asset    | balance |
      | buyer1  | USDT.0.1 | 5000    |
      | seller1 | USDT.0.1 | 0       |
      | buyer2  | USDT.0.1 | 0       |
      | seller2 | USDT.0.1 | 5000    |


  Scenario: Given a liquidity fees received reward, contributions from a spot market are correctly aggregated with markets allowing leverage. (0056-REWA-167). Given the following dispatch metrics, if no `eligible keys` list is specified in the recurring transfer, all parties meeting other eligibility criteria should receive a score 0056-REWA-202.
    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED | USDT.0.1     |         | 100         |
    And the network moves ahead "1" epochs

    # Generate trades on the spot market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller liquidity fee |
      | buyer1 | seller1 | 1000  | 10   | 0               | 100                  |

    Given clear trade events

    # Generate trades on the leveraged market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | GOLD/USDT | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | GOLD/USDT | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer liquidity fee | seller maker fee |
      | buyer2 | seller2 | 1000  | 10   | 100                 | 0                |

    # Move to the end of the epoch - LP1 and LP2 should receive equal
    # rewards as they received an equal amount of liquidity fees from
    # the spot and leveraged markets respectively.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party | asset    | balance |
      | lp1   | USDT.0.1 | 5000    |
      | lp2   | USDT.0.1 | 5000    |

  Scenario: Given the following dispatch metrics, if an `eligible keys` list is specified in the recurring transfer, only parties included in the list and meeting other eligibility criteria should receive a score 0056-REWA-213.
    # Set-up a recurring transfer
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | eligible_keys |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED | USDT.0.1     |         | 100         | lp1           |
    And the network moves ahead "1" epochs

    # Generate trades on the spot market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | buyer1  | BTC/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | seller1 | BTC/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer maker fee | seller liquidity fee |
      | buyer1 | seller1 | 1000  | 10   | 0               | 100                  |

    Given clear trade events

    # Generate trades on the leveraged market
    Given the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | seller2 | GOLD/USDT | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buyer2  | GOLD/USDT | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller  | price | size | buyer liquidity fee | seller maker fee |
      | buyer2 | seller2 | 1000  | 10   | 100                 | 0                |

    # Move to the end of the epoch - LP1 and LP2 should receive equal but only lp1 is in eligible keys so they get it all
    # rewards as they received an equal amount of liquidity fees from
    # the spot and leveraged markets respectively.
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party | asset    | balance |
      | lp1   | USDT.0.1 | 10000    |
