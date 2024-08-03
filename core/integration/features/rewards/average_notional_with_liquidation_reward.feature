Feature: Calculation of average position during closeout trades

    Background:

        # Initialise the network
        Given time is updated to "2023-01-01T00:00:00Z"
        And the average block duration is "1"
        And the following network parameters are set:
            | name                                    | value |
            | market.fee.factors.makerFee             | 0.001 |
            | network.markPriceUpdateMaximumFrequency | 0s    |
            | market.auction.minimumDuration          | 1     |
            | validators.epoch.length                 | 60s   |
            | limits.markets.maxPeggedOrders          | 4     |
            | referralProgram.minStakedVegaTokens     | 0     |
            | rewards.vesting.baseRate                | 1.0   |


        # Initialise the markets
        And the following assets are registered:
            | id       | decimal places | quantum |
            | USD-1-10 | 1              | 10      |
        And the markets:
            | id           | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places |
            | ETH/USD-1-10 | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       |

        # Initialise the parties
        Given the parties deposit on asset's general account the following amount:
            | party                                                            | asset    | amount      |
            | lpprov                                                           | USD-1-10 | 10000000000 |
            | aux1                                                             | USD-1-10 | 10000000    |
            | aux2                                                             | USD-1-10 | 10000000    |
            | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 10000000000 |
            | party1                                                           | USD-1-10 | 1000        |

        # Exit opening auctions
        Given the parties submit the following liquidity provision:
            | id  | party  | market id    | commitment amount | fee  | lp type    |
            | lp1 | lpprov | ETH/USD-1-10 | 1000000           | 0.01 | submission |
        And the parties place the following pegged iceberg orders:
            | party  | market id    | peak size | minimum visible size | side | pegged reference | volume | offset |
            | lpprov | ETH/USD-1-10 | 5000      | 1000                 | buy  | BID              | 10000  | 10     |
            | lpprov | ETH/USD-1-10 | 5000      | 1000                 | sell | ASK              | 10000  | 10     |
        When the parties place the following orders:
            | party | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/USD-1-10 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/USD-1-10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
        And the opening auction period ends for market "ETH/USD-1-10"
        When the network moves ahead "1" blocks
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"

    @Liquidation
    Scenario: Bug time-weighted average position not updated correctly during closeout trades
        # Setup such that distributed rewards are all vested the following epoch, i,e. the balance in the vested account is equal to the rewards distributed that epocha

        # Close open positions to simplify test
        When the parties place the following orders:
            | party | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux2  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/USD-1-10 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        Then the network moves ahead "1" epochs

        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                      | asset    | amount | start_epoch | end_epoch | factor | metric                           | metric_asset | markets | lock_period | window_length | distribution_strategy | entity_scope | individual_scope | staking_requirement | notional_requirement | ranks        |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL | USD-1-10 | 10000  | 2           |           | 1      | DISPATCH_METRIC_AVERAGE_NOTIONAL | USD-1-10     |         | 1           | 1             | RANK                  | INDIVIDUALS  | ALL              | 0                   | 0                    | 1:10,2:5,4:1 |
        When the parties place the following orders:
            | party  | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux1   | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | party1 | ETH/USD-1-10 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        Then the network moves ahead "1" epochs
        Then parties should have the following vesting account balances:
            | party  | asset    | balance |
            | party1 | USD-1-10 | 5000    |

        # Move into the epoch then move the mark price triggering a closeout
        Given the network moves ahead "5" blocks
        And the parties place the following orders:
            | party | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux2  | ETH/USD-1-10 | buy  | 1      | 1099  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/USD-1-10 | sell | 1      | 1099  | 1                | TYPE_LIMIT | TIF_GTC |
        When the network moves ahead "2" blocks
        # Trades should result in all parties having no open position
        Then the following trades should be executed:
            | buyer   | price | size | seller  | aggressor side |
            | party1  | 1099  | 1    | network | sell           |
            | network | 1100  | 1    | aux2    | buy            |
        And the parties should have the following profit and loss:
            | party  | volume | unrealised pnl | realised pnl |
            | party1 | 0      | 0              | -890         |
            | aux1   | 0      | 0              | 890          |
            | aux2   | 0      | 0              | 0            |
            | lpprov | 0      | 0              | 0            |
        # Expect to see rewards as positions open at the start of the epoch
        Then parties should have the following vesting account balances:
            | party  | asset    | balance |
            | party1 | USD-1-10 | 5000    |
            | aux1   | USD-1-10 | 5000    |
        Given the network moves ahead "1" epochs
        # At the beginning of the epoch the party had some position so they still get some reward at this epoch
        Then parties should have the following vesting account balances:
            | party  | asset    | balance |
            | party1 | USD-1-10 | 3846    |
            | aux1   | USD-1-10 | 3846    |
            | aux2   | USD-1-10 | 1923    |

        Given the network moves ahead "1" epochs
        # there are still rewards because while the position is 0 at the beginning of the epoch, the timeweighted position will only become 0
        # at the beginning of the next epoch
        Then parties should have the following vesting account balances:
            | party  | asset    | balance |
            | party1 | USD-1-10 | 3333    |
            | aux2   | USD-1-10 | 3333    |
            | aux1   | USD-1-10 | 3333    |

        Given the network moves ahead "1" epochs
        # Expect to see no rewards as no positions open at the start of the epoch
        Then parties should have the following vesting account balances:
            | party  | asset    | balance |
            | party1 | USD-1-10 | 0       |
            | aux2   | USD-1-10 | 0       |
            | aux1   | USD-1-10 | 0       |


