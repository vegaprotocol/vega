Feature: Allow markets to be specified with a smaller number of decimal places than the underlying settlement asset

    Background:
        Given the following network parameters are set:
            | name                                          | value |
            | market.stake.target.timeWindow                | 24h   |
            | market.stake.target.scalingFactor             | 1     |
            | market.liquidity.bondPenaltyParameter         | 0.2   |
            | market.liquidity.targetstake.triggering.ratio | 0.1   |
        And the following assets are registered:
            | id  | decimal places |
            | ETH | 5              |
            | USD | 2              |
        And the average block duration is "1"
        And the log normal risk model named "log-normal-risk-model-1":
            | risk aversion | tau | mu | r | sigma |
            | 0.000001      | 0.1 | 0  | 0 | 1.0   |
        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.004     | 0.001              |
        And the price monitoring updated every "1" seconds named "price-monitoring-1":
            | horizon | probability | auction extension |
            | 1       | 0.99        | 300               |
        And the markets:
            | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | decimal places | position decimal places |
            | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       |
            | USD/DEC19 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | default-none       | default-usd-for-future | 3              | 3                       |
            | USD/DEC20 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | default-none       | default-usd-for-future | 5              | 5                       |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount    |
            | party0 | USD   | 5000000   |
            | party0 | ETH   | 5000000   |
            | party1 | USD   | 100000000 |
            | party1 | ETH   | 100000000 |
            | party2 | USD   | 100000000 |
            | party2 | ETH   | 100000000 |
            | party3 | USD   | 100000000 |

    Scenario: Users engage in a USD market auction, 0070-MKTD-003
        Given the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
            | lp1 | party0 | ETH/MAR22 | 500               | 0.001 | sell | ASK              | 500        | 20     | submission |
            | lp1 | party0 | ETH/MAR22 | 500               | 0.001 | buy  | BID              | 500        | -20    | amendment  |

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
            | party1 | ETH/MAR22 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | ETH/MAR22 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | ETH/MAR22 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
            | party2 | ETH/MAR22 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
            | party2 | ETH/MAR22 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
            | party2 | ETH/MAR22 | sell | 1      | 11    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

        When the opening auction period ends for market "ETH/MAR22"
        Then the auction ends with a traded volume of "10" at a price of "10"
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond |
            | party0 | USD   | ETH/MAR22 | 1922   | 4997578  | 500  |
            | party1 | USD   | ETH/MAR22 | 12730  | 99987270 | 0    |
            | party2 | USD   | ETH/MAR22 | 51819  | 99948181 | 0    |
        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 10    | 10   | party2 |

    Scenario: Users engage in an ETH market auction, 0070-MKTD-003
        Given the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
            | lp1 | party0 | USD/DEC19 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
            | lp1 | party0 | USD/DEC19 | 50000             | 0.001 | buy  | BID              | 500        | -20    | amendment  |

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
            | party1 | USD/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | USD/DEC19 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | USD/DEC19 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
            | party2 | USD/DEC19 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
            | party2 | USD/DEC19 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
            | party2 | USD/DEC19 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

        When the opening auction period ends for market "USD/DEC19"
        Then the auction ends with a traded volume of "10" at a price of "1000"
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond  |
            | party0 | ETH   | USD/DEC19 | 853657 | 4096343  | 50000 |
            | party1 | ETH   | USD/DEC19 | 1273   | 99998727 | 0     |
            | party2 | ETH   | USD/DEC19 | 5188   | 99994812 | 0     |
        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 1000  | 10   | party2 |

    Scenario: Users engage in an ETH market auction with full decimal places, 0070-MKTD-003

        Given  the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
            | lp1 | party0 | USD/DEC20 | 500               | 0.001 | sell | ASK              | 500        | 20     | submission |
            | lp1 | party0 | USD/DEC20 | 500               | 0.001 | buy  | BID              | 500        | -20    | amendment  |

        And the parties place the following orders:
            | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference  |
            | party1 | USD/DEC20 | buy  | 1      | 90000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | USD/DEC20 | buy  | 1      | 90000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | USD/DEC20 | buy  | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
            | party2 | USD/DEC20 | sell | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
            | party2 | USD/DEC20 | sell | 1      | 101000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
            | party2 | USD/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

        When the opening auction period ends for market "USD/DEC20"
        Then the auction ends with a traded volume of "10" at a price of "100000"
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond |
            | party0 | ETH   | USD/DEC20 | 426829 | 4572671  | 500  |
            | party1 | ETH   | USD/DEC20 | 13     | 99999987 | 0    |
            | party2 | ETH   | USD/DEC20 | 52     | 99999948 | 0    |
        And the following trades should be executed:
            | buyer  | price  | size | seller |
            | party1 | 100000 | 10   | party2 |

    Scenario: User tops up markets with differing precisions with the same asset + amount, should result in identical margin changes, 0070-MKTD-004

        Given  the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
            | lp1 | party0 | USD/DEC20 | 100000            | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp1 | party0 | USD/DEC20 | 100000            | 0.001 | buy  | BID              | 100        | -20    | amendment  |
            | lp2 | party0 | USD/DEC19 | 1000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp2 | party0 | USD/DEC19 | 1000              | 0.001 | buy  | BID              | 100        | -20    | amendment  |

        And the parties place the following orders:
            | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference   |
            | party1 | USD/DEC20 | buy  | 1      | 90000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1   |
            | party1 | USD/DEC20 | buy  | 1      | 90000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1   |
            | party1 | USD/DEC20 | buy  | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2   |
            | party2 | USD/DEC20 | sell | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3  |
            | party2 | USD/DEC20 | sell | 1      | 101000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1  |
            | party2 | USD/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2  |
            | party1 | USD/DEC19 | buy  | 1      | 900    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1a  |
            | party1 | USD/DEC19 | buy  | 1      | 900    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1a  |
            | party1 | USD/DEC19 | buy  | 10     | 1000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2a  |
            | party2 | USD/DEC19 | sell | 10     | 1000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3a |
            | party2 | USD/DEC19 | sell | 1      | 1010   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1a |
            | party2 | USD/DEC19 | sell | 1      | 1100   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2a |

        When the opening auction period ends for market "USD/DEC20"
        And the opening auction period ends for market "USD/DEC19"
        Then the parties should have the following account balances:
            | party  | asset | market id | margin  | general  | bond   |
            | party0 | ETH   | USD/DEC20 | 1280486 | 3191685  | 100000 |
            | party1 | ETH   | USD/DEC20 | 13      | 99998714 | 0      |
            | party2 | ETH   | USD/DEC20 | 52      | 99994760 | 0      |
            | party0 | ETH   | USD/DEC19 | 426829  | 3191685  | 1000   |
            | party1 | ETH   | USD/DEC19 | 1273    | 99998714 | 0      |
            | party2 | ETH   | USD/DEC19 | 5188    | 99994760 | 0      |
