Feature: Realised returns reward metric

  Background:

    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | validators.epoch.length                 | 60s   |
      | market.fee.factors.makerFee             | 0.1   |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    # Setup market and parties
    Given the following assets are registered:
      | id       | decimal places | quantum |
      | USDT.0.1 | 0              | 1       |
    And the parties deposit on asset's general account the following amount:
      | party                                                            | asset    | amount     |
      | aux1                                                             | USDT.0.1 | 1000000000 |
      | aux2                                                             | USDT.0.1 | 1000000000 |
      | party1                                                           | USDT.0.1 | 1000000000 |
      | party2                                                           | USDT.0.1 | 1000000000 |
      | party3                                                           | USDT.0.1 | 1000000000 |
      | party4                                                           | USDT.0.1 | 1000000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USDT.0.1 | 1000000000 |
    And the markets:
      | id       | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USDT | USDT.0.1   | USDT.0.1 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |

    # Exit opening auction and close the positions of the auxiliary parties so they don't receive rewards
    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDT  | buy  | 1000   | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | sell | 1000   | 1999  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/USDT"
    And the parties should have the following profit and loss:
      | party | volume | unrealised pnl | realised pnl |
      | aux1  | 1      | 0              | 0            |
      | aux2  | -1     | 0              | 0            |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USDT"
    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDT  | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    And the parties should have the following profit and loss:
      | party | volume | unrealised pnl | realised pnl |
      | aux1  | 0      | 0              | 0            |
      | aux2  | 0      | 0              | 0            |


  #   Scenario: In a falling market where short positions are profitable, parties have only unrealised pnl (0056-REWA-118)(0056-REWA-129)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     And the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 10     | -1000          | 0            |
  #       | party2 | -10    | 1000           | 0            |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 0       |
  #       | party2 | USDT.0.1 | 0       |


  #   Scenario: In a falling market where short positions are profitable, parties have party realised pnl (0056-REWA-119)(0056-REWA-130)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | sell | 5      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | buy  | 5      | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 5      | -500           | -500         |
  #       | party2 | -5     | 500            | 500          |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 0       |
  #       | party2 | USDT.0.1 | 10000   |


  #   Scenario: In a falling market where short positions are profitable, parties have fully realised pnl (0056-REWA-120)(0056-REWA-131)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | sell | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | buy  | 10     | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 0      | 0              | -1000        |
  #       | party2 | 0      | 0              | 1000         |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 0       |
  #       | party2 | USDT.0.1 | 10000   |


  #   Scenario: In a falling market where short positions are profitable, parties fully realise pnl and switch sides (0056-REWA-132)(0056-REWA-135)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | party3 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party4 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | sell | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | buy  | 10     | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #       | party3 | ETH/USDT  | sell | 15     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party4 | ETH/USDT  | buy  | 15     | 900   | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 0      | 0              | -1000        |
  #       | party2 | 0      | 0              | 1000         |
  #       | party3 | -5     | 0              | -1000        |
  #       | party4 | 5      | 0              | 1000         |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 0       |
  #       | party2 | USDT.0.1 | 5000    |
  #       | party3 | USDT.0.1 | 0       |
  #       | party4 | USDT.0.1 | 5000    |


  #   Scenario: In a rising market where long positions are profitable, parties have only unrealised pnl (0056-REWA-122)(0056-REWA-125)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     And the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 10     | 1000           | 0            |
  #       | party2 | -10    | -1000          | 0            |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 0       |
  #       | party2 | USDT.0.1 | 0       |


  #   Scenario: In a rising market where long positions are profitable, parties have party realised pnl (0056-REWA-123)(0056-REWA-126)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | sell | 5      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | buy  | 5      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 5      | 500            | 500          |
  #       | party2 | -5     | -500           | -500         |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 10000   |
  #       | party2 | USDT.0.1 | 0       |


  #   Scenario: In a rising market where long positions are profitable, parties have fully realised pnl (0056-REWA-124)(0056-REWA-127)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | buy  | 10     | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 0      | 0              | 1000         |
  #       | party2 | 0      | 0              | -1000        |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 10000   |
  #       | party2 | USDT.0.1 | 0       |


  #   Scenario: In a rising market where long positions are profitable, parties fully realise pnl and switch sides (0056-REWA-133)(0056-REWA-134)

  #     # Set-up a recurring transfer dispatching rewards based on realised profit and loss
  #     Given the current epoch is "0"
  #     And the parties submit the following recurring transfers:
  #       | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
  #       | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
  #     And the network moves ahead "1" epochs

  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | party3 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party4 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | aux1   | ETH/USDT  | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | aux2   | ETH/USDT  | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     Given the parties place the following orders:
  #       | party  | market id | side | volume | price | resulting trades | type       | tif     |
  #       | party1 | ETH/USDT  | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party2 | ETH/USDT  | buy  | 10     | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #       | party3 | ETH/USDT  | sell | 15     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
  #       | party4 | ETH/USDT  | buy  | 15     | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
  #     When the network moves ahead "1" blocks
  #     Then the parties should have the following profit and loss:
  #       | party  | volume | unrealised pnl | realised pnl |
  #       | party1 | 0      | 0              | 1000         |
  #       | party2 | 0      | 0              | -1000        |
  #       | party3 | -5     | 0              | 1000         |
  #       | party4 | 5      | 0              | -1000        |
  #       | aux1   | 1      | 0              | 0            |
  #       | aux2   | -1     | 0              | 0            |
  #     # Move to the end of the epoch
  #     Given the network moves ahead "1" epochs
  #     Then parties should have the following vesting account balances:
  #       | party  | asset    | balance |
  #       | party1 | USDT.0.1 | 5000    |
  #       | party2 | USDT.0.1 | 0       |
  #       | party3 | USDT.0.1 | 5000    |
  #       | party4 | USDT.0.1 | 0       |


  Scenario: Parties with long and short positions open and close position at same price, they have 0 realised returns but should still receive rewards (0056-REWA-121)(0056-REWA-128)(0056-REWA-210)
    # Set-up a recurring transfer dispatching rewards based on realised profit and loss
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/USDT  | buy  | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/USDT  | sell | 10     | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 0            |
      | party2 | 0      | 0              | 0            |
      | party3 | 0      | 0              | -1000        |
      | party4 | 0      | 0              | 1000         |
    # Move to the end of the epoch
    Given the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party  | asset    | balance |
      | party1 | USDT.0.1 | 2500    |
      | party2 | USDT.0.1 | 2500    |
      | party3 | USDT.0.1 | 0       |
      | party4 | USDT.0.1 | 5000    |

  Scenario: Given the following dispatch metrics, if an `eligible keys` list is specified in the recurring transfer, only parties included in the list and meeting other eligibility criteria should receive a score (0056-REWA-220)
    # Set-up a recurring transfer dispatching rewards based on realised profit and loss
    Given the current epoch is "0"
    And the parties submit the following recurring transfers:
      | id     | from                                                             | from_account_type    | to                                                               | to_account_type                     | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets  | lock_period | eligible_keys |
      | reward | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_REALISED_RETURN | USDT.0.1 | 10000  | 1           |           | 1      | DISPATCH_METRIC_REALISED_RETURN | USDT.0.1     | ETH/USDT | 100         | party1,party3 |
    And the network moves ahead "1" epochs

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/USDT  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USDT  | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/USDT  | buy  | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/USDT  | sell | 10     | 1100  | 1                | TYPE_LIMIT | TIF_GTC |
    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/USDT  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/USDT  | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 0      | 0              | 0            |
      | party2 | 0      | 0              | 0            |
      | party3 | 0      | 0              | -1000        |
      | party4 | 0      | 0              | 1000         |
    # Move to the end of the epoch
    Given the network moves ahead "1" epochs
    # party 3 has negative score, party1 has 0 score so party take the whole lot as only they are in the eligible keys. 
    Then parties should have the following vesting account balances:
      | party  | asset    | balance |
      | party1 | USDT.0.1 | 10000   |
      | party2 | USDT.0.1 | 0       |
      | party3 | USDT.0.1 | 0       |
      | party4 | USDT.0.1 | 0       |