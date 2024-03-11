Feature: Test internal and external twap calculation

    Background:
        # epoch time is 1602806400
        Given time is updated to "2020-10-16T00:00:00Z"
        And the perpetual oracles from "0xCAFECAFE1":
            | name        | asset | settlement property | settlement type | schedule property | schedule type  | funding rate scaling factor | quote name | settlement decimals |
            | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 2.5                         | ETH        | 18                  |

        And the perpetual oracles from "0xCAFECAFE1":
            | name        | asset | settlement property | settlement type | schedule property | schedule type   | funding rate lower bound | funding rate upper bound | quote name | settlement decimals |
            | perp-oracle2 | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | -0.005                   | 0.005                    | ETH        | 18                  |

        And the liquidity sla params named "SLA":
            | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
            | 100.0       | 0.5                          | 1                             | 1.0                    |
        And the price monitoring named "my-price-monitoring":
            | horizon | probability | auction extension |
            | 43200   | 0.9999999   | 120               |
        And the log normal risk model named "my-log-normal-risk-model":
            | risk aversion | tau                    | mu | r     | sigma |
            | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 0.8   |
        And the markets:
            | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params |
            | ETH/DEC19 | ETH        | USD   | my-log-normal-risk-model | default-margin-calculator | 120              | default-none | my-price-monitoring | perp-oracle        | 0.25                   | 0                         | -3                      | perp        | SLA        |
        And the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 0s    |
        And the average block duration is "1"
        When the parties deposit on asset's general account the following amount:
            | party  | asset | amount       |
            | party1 | USD   | 100000000000 |
            | party2 | USD   | 100000000000 |
            | party3 | USD   | 100000000000 |
            | aux    | USD   | 100000000000 |
            | aux2   | USD   | 100000000000 |
            | lpprov | USD   | 100000000000 |
        Then the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | lp type    |
            | lp1 | lpprov | ETH/DEC19 | 100000            | 0.001 | submission |
        # move market to continuous
        And the parties place the following orders:
            | party  | market id | side | volume | price   | resulting trades | type       | tif     |
            | aux2   | ETH/DEC19 | buy  | 1      | 1       | 0                | TYPE_LIMIT | TIF_GTC |
            | lpprov | ETH/DEC19 | buy  | 100000 | 1       | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2   | ETH/DEC19 | buy  | 1      | 1000    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 1000    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 3500    | 0                | TYPE_LIMIT | TIF_GTC |
            | lpprov | ETH/DEC19 | sell | 100000 | 3500    | 0                | TYPE_LIMIT | TIF_GTC |
        And the market data for the market "ETH/DEC19" should be:
            | target stake  | supplied stake |
            | 431500         | 100000         |
        And the opening auction period ends for market "ETH/DEC19"

    @Perpetual @funding-rate
    Scenario: 0053-PERP-029,  0053-PERP-030 Funding rate modified by perpetual parameters
        Given the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
        And time is updated to "2020-10-16T00:05:00Z"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 990    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 990    | 1                | TYPE_LIMIT | TIF_GTC |
        Then time is updated to "2020-10-16T00:10:00Z"

        # 1602806400 + 120s = 1602807000
        # funding period is ended with perp.funding.cue
        And the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name             | value                  | time offset |
            | perp.ETH.value   | 1000000000000000000000 | -1s         |
            | perp.funding.cue | 1602807000             | 0s          |
        And the mark price should be "990" for the market "ETH/DEC19"

        # 0053-PERP-029 funding rate will be 2.5 times what it would normally be given funding-rate-scaling-factor
        Then the product data for the market "ETH/DEC19" should be:
            | internal twap | external twap  | funding payment | funding rate |
            | 990           | 1000           | -25             | -0.025       |

        # update to the perp spec with the lower bound on the funding rate
        And the markets are updated:
          | id        | data source config | linear slippage factor | quadratic slippage factor |
          | ETH/DEC19 | perp-oracle2       | 1e-3                   | 0                         |

        Then time is updated to "2020-10-16T00:10:01Z"

        # 0053-PERP-030 funding rate will be snapped to -0.005 since it should *really* be -0.01, but thats lower than funding-rate-lower-bound
        Then the product data for the market "ETH/DEC19" should be:
            | internal twap | external twap  | funding payment | funding rate |
            | 990           | 1000           | -5              | -0.005       |
        



