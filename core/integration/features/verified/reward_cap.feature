Feature: Capping distributed rewards at a multiple of fees paid

  Tests check the cap_reward_fee_multiple correctly caps rewards distributed
  to parties to a multiple of the total fees paid by each respective party.


  Background:

    # Initialise the network and register the assets
    Given time is updated to "2023-01-01T00:00:00Z"
    And the average block duration is "1"
    And the following network parameters are set:
      | name                                 | value  |
      | market.fee.factors.makerFee          | 0.0005 |
      | market.fee.factors.infrastructureFee | 0.0015 |
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD-1-10 | 1              | 10      |

    # Initialise the parties and deposit assets
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset    | amount      |
      | lp                                                               | USD-1-10 | 10000000000 |
      | aux1                                                             | USD-1-10 | 10000000000 |
      | aux2                                                             | USD-1-10 | 10000000000 |
      | party1                                                           | USD-1-10 | 10000000000 |
      | party2                                                           | USD-1-10 | 10000000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 10000000000 |

    # Setup the market in continuous trading
    And the markets:
      | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
      | ETH/USD-1-10 | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |
    Given the parties submit the following liquidity provision:
      | id  | party | market id    | commitment amount | fee   | lp type    |
      | lp1 | lp    | ETH/USD-1-10 | 1000000           | 0.008 | submission |
    And the parties place the following orders:
      | party | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USD-1-10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/USD-1-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"

  Scenario:

    # Start a new epoch then generate trades
    Given the network moves ahead "1" epochs
    And the parties place the following orders:
      | party  | market id    | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/USD-1-10 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/USD-1-10 | buy  | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | aux1   | ETH/USD-1-10 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/USD-1-10 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | price | size | seller | buyer fee | seller fee | buyer maker fee | seller maker fee | buyer infrastructure fee | seller infrastructure fee | buyer liquidity fee | seller liquidity fee |
      | party1 | 1000  | 10   | aux1   | 1000      | 0          | 50              | 0                | 150                      | 0                         | 800                 | 0                    |
      | aux1   | 1000  | 20   | party2 | 0         | 2000       | 0               | 100              | 0                        | 300                       | 0                   | 1600                 |

    
    # Setup a recurring transfer funding a reward pool
    Given the current epoch is "1"
    And the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | entity_scope | individual_scope | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | cap_reward_fee_multiple | lock_period |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | INDIVIDUALS  | ALL              | USD-1-10 | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 1                       | 100         |
    When the network moves ahead "1" epochs
    Then parties should have the following vesting account balances:
      | party  | asset    | balance |
      | party1 | USD-1-10 | 200     |
      | party2 | USD-1-10 | 400     |
