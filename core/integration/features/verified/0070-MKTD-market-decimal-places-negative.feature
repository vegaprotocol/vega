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
        #risk factor short: 3.5569036
        #risk factor long: 0.801225765

        And the fees configuration named "fees-config-1":
            | maker fee | infrastructure fee |
            | 0.004     | 0.001              |
        And the price monitoring named "price-monitoring-1":
            | horizon | probability | auction extension |
            | 1       | 0.99        | 300               |
        And the markets:
            | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | decimal places | position decimal places |
            | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       |
            | USD/DEC19 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 3              | 3                       |
            | USD/DEC20 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 5                       |
            | USD/DEC21 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | 3                       |
            | USD/DEC22 | USD        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none  | price-monitoring-1 | default-usd-for-future | 5              | -1                      |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount    |
            | party0 | USD   | 5000000   |
            | party0 | ETH   | 5000000   |
            | party1 | USD   | 100000000 |
            | party1 | ETH   | 100000000 |
            | party2 | USD   | 100000000 |
            | party2 | ETH   | 100000000 |
            | party3 | USD   | 100000000 |
            | lpprov | ETH   | 100000000 |
            | lpprov | USD   | 100000000 |

    Scenario: 001, Markets with different precisions trade at the same price

        Given  the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
            | lp0 | party0 | USD/DEC20 | 1000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp0 | party0 | USD/DEC20 | 1000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp1 | party0 | USD/DEC21 | 1000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp1 | party0 | USD/DEC21 | 1000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp2 | party0 | USD/DEC19 | 1000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp2 | party0 | USD/DEC19 | 1000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp7 | party0 | USD/DEC22 | 1000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp7 | party0 | USD/DEC22 | 1000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp3 | lpprov | USD/DEC20 | 4000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp3 | lpprov | USD/DEC20 | 4000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp4 | lpprov | USD/DEC21 | 4000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp4 | lpprov | USD/DEC21 | 4000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp5 | lpprov | USD/DEC19 | 4000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp5 | lpprov | USD/DEC19 | 4000              | 0.001 | buy  | BID              | 100        | -20    | submission |
            | lp6 | lpprov | USD/DEC22 | 4000              | 0.001 | sell | ASK              | 100        | 20     | submission |
            | lp6 | lpprov | USD/DEC22 | 4000              | 0.001 | buy  | BID              | 100        | -20    | submission |

        And the parties place the following orders:
            | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference   |
            | party1 | USD/DEC21 | buy  | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2   |
            | party2 | USD/DEC21 | sell | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3  |
            | party1 | USD/DEC20 | buy  | 1000   | 100000 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2   |
            | party2 | USD/DEC20 | sell | 1000   | 100000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3  |
            | party1 | USD/DEC19 | buy  | 10     | 1000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2a  |
            | party2 | USD/DEC19 | sell | 10     | 1000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3a |
            | party1 | USD/DEC22 | buy  | 1      | 1000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2a  |
            | party2 | USD/DEC22 | sell | 1      | 1000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3a |
            | party0 | USD/DEC21 | buy  | 1      | 90000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1   |
            | party0 | USD/DEC21 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2  |
            | party0 | USD/DEC20 | buy  | 1      | 90000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1   |
            | party0 | USD/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2  |
            | party0 | USD/DEC19 | buy  | 1      | 900    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1a  |
            | party0 | USD/DEC19 | sell | 1      | 1100   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2a |
            | party0 | USD/DEC22 | buy  | 1      | 900    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1a  |
            | party0 | USD/DEC22 | sell | 1      | 1100   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-4a |

        Then the market data for the market "USD/DEC19" should be:
            | target stake | supplied stake |
            | 3556         | 5000           |
        Then the market data for the market "USD/DEC20" should be:
            | target stake | supplied stake |
            | 3556         | 5000           |
        Then the market data for the market "USD/DEC21" should be:
            | target stake | supplied stake |
            | 3556         | 5000           |
        Then the market data for the market "USD/DEC22" should be:
            | target stake | supplied stake |
            | 35569        | 5000           |
        # target stake= vol * mark price * rf = 1*1000*3.5569036*10 = 35569

        When the opening auction period ends for market "USD/DEC21"
        And the opening auction period ends for market "USD/DEC20"
        And the opening auction period ends for market "USD/DEC19"
        And the opening auction period ends for market "USD/DEC22"

        Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USD/DEC21"
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USD/DEC20"
        And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USD/DEC19"
        And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "USD/DEC22"

        Then the parties should have the following account balances:
            | party  | asset | market id | margin | general  | bond |
            | party0 | ETH   | USD/DEC21 | 8536   | 4881525  | 1000 |
            | party1 | ETH   | USD/DEC21 | 1081   | 99977539 | 0    |
            | party2 | ETH   | USD/DEC21 | 4388   | 99901468 | 0    |
            | party0 | ETH   | USD/DEC20 | 7768   | 4881525  | 1000 |
            | party1 | ETH   | USD/DEC20 | 1081   | 99977539 | 0    |
            | party2 | ETH   | USD/DEC20 | 4388   | 99901468 | 0    |
            | party0 | ETH   | USD/DEC19 | 8536   | 4881525  | 1000 |
            | party1 | ETH   | USD/DEC19 | 1081   | 99977539 | 0    |
            | party2 | ETH   | USD/DEC19 | 4388   | 99901468 | 0    |
            | party0 | ETH   | USD/DEC22 | 89635  | 4881525  | 1000 |
            | party1 | ETH   | USD/DEC22 | 19218  | 99977539 | 0    |
            | party2 | ETH   | USD/DEC22 | 85368  | 99901468 | 0    |


    Scenario: 002, Users engage in a USD market auction, (0070-MKTD-003, 0070-MKTD-008)
        Given the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
            | lp1 | party0 | USD/DEC22 | 35569             | 0.001 | sell | ASK              | 500        | 20     | submission |
            | lp1 | party0 | USD/DEC22 | 35569             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
            | party1 | USD/DEC22 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | USD/DEC22 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
            | party1 | USD/DEC22 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
            | party2 | USD/DEC22 | sell | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
            | party2 | USD/DEC22 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
            | party2 | USD/DEC22 | sell | 1      | 11    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

        When the opening auction period ends for market "USD/DEC22"
        Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "USD/DEC22"
        And the auction ends with a traded volume of "10" at a price of "10"
        And the parties should have the following account balances:
            | party  | asset | market id | margin | general   | bond  |
            | party0 | USD   | USD/DEC22 | 303902 | 5000000   | 35569 |
            | party1 | USD   | USD/DEC22 | 1273   | 100000000 | 0     |
            | party2 | USD   | USD/DEC22 | 5122   | 100000000 | 0     |

        And the parties should have the following margin levels:
            | party  | market id | maintenance | search | initial | release |
            | party0 | USD/DEC22 | 253252      | 278577 | 303902  | 354552  |
            | party1 | USD/DEC22 | 1061        | 1167   | 1273    | 1485    |
            | party2 | USD/DEC22 | 4269        | 4695   | 5122    | 5976    |

        #margin for party0 = 712*10*10*3.5569036+791*10*10*0.801225765=316629
        And the following trades should be executed:
            | buyer  | price | size | seller |
            | party1 | 10    | 10   | party2 |

        Then the order book should have the following volumes for market "USD/DEC22":
            | side | price | volume |
            | sell | 11    | 1      |
            | sell | 10    | 713    |
            | buy  | 9     | 793    |
            | buy  | 10    | 0      |
# LP vol_sell=  35569/10/10/0.5=712
# LP vol_buy=  35569/9/10/0.5=791, and there are 2 existing buy order on the book which makes it 793


