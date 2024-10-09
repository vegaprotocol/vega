Feature: Team Rewards

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
            | aux3                                                             | USD-1-10 | 10000000000 |
            | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USD-1-10 | 10000000000 |
            | party1                                                           | USD-1-10 | 10000000000 |
            | party2                                                           | USD-1-10 | 10000000000 |
            | party3                                                           | USD-1-10 | 10000000000 |

        # Exit opening auctions
        Given the parties submit the following liquidity provision:
            | id  | party  | market id    | commitment amount | fee  | lp type    |
            | lp1 | lpprov | ETH/USD-1-10 | 1000000           | 0.01 | submission |
        And the parties place the following pegged iceberg orders:
            | party  | market id    | peak size | minimum visible size | side | pegged reference | volume | offset |
            | lpprov | ETH/USD-1-10 | 5000      | 1000                 | buy  | BID              | 10000  | 1      |
            | lpprov | ETH/USD-1-10 | 5000      | 1000                 | sell | ASK              | 10000  | 1      |
        When the parties place the following orders:
            | party | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/USD-1-10 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/USD-1-10 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/USD-1-10 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
        And the opening auction period ends for market "ETH/USD-1-10"
        When the network moves ahead "1" blocks
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/USD-1-10"

        # Create a team
        And the following teams with referees are created:
            | referrer  | prefix | code            | team name | referees | balance  | asset    |
            | referrer1 | ref1   | referral-code-1 | team1     | 3        | 10000000 | USD-1-10 |

    @TeamStep
    Scenario: Given a recurring transfer with a reward window `>1` and specifying a non-zero `target_notional_volume`. When the volume is spread evenly across the window, if the cumulative volume across the window is less than the target, the rewards distributed are scaled accordingly. (<a name="0056-REWA-240" href="#0056-REWA-240">0056-REWA-240</a>).
        And the parties place the following orders:
            | party     | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux1      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | party1    | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
            | aux1      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | party2    | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
            | aux3      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | ref1-0001 | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
            | aux3      | ETH/USD-1-10 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | ref1-0002 | ETH/USD-1-10 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        When the network moves ahead "1" epochs
        Given the parties submit the following recurring transfers:
            | id | from                                                             | from_account_type    | to                                                               | to_account_type                     | entity_scope | individual_scope | asset    | amount | start_epoch | end_epoch | factor | metric                          | metric_asset | markets      | target_notional_volume | window_length |
            | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES | INDIVIDUALS  | ALL              | USD-1-10 | 10000  | 2           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID | USD-1-10     | ETH/USD-1-10 | 1000000                | 2             |
        And the parties place the following orders:
            | party     | market id    | side | volume | price | resulting trades | type       | tif     |
            | aux1      | ETH/USD-1-10 | sell | 9      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | party1    | ETH/USD-1-10 | buy  | 9      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
            | aux1      | ETH/USD-1-10 | sell | 9      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | party2    | ETH/USD-1-10 | buy  | 9      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
            | aux3      | ETH/USD-1-10 | sell | 9      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | ref1-0001 | ETH/USD-1-10 | buy  | 9      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
            | aux3      | ETH/USD-1-10 | sell | 9      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
            | ref1-0002 | ETH/USD-1-10 | buy  | 9      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
        When the network moves ahead "1" epochs

        # notional is uneven across the window, but total notional traded over the 2 epochs is 40% of the target notional
        # so the reward paid is 4000 split equally between parties.
        Then parties should have the following vesting account balances:
            | party     | asset    | balance |
            | party1    | USD-1-10 | 1000    |
            | party2    | USD-1-10 | 1000    |
            | ref1-0001 | USD-1-10 | 1000    |
            | ref1-0002 | USD-1-10 | 1000    |
