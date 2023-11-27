Feature: Test internal and external twap calculation

    Background:
        # epoch time is 1602806400
        Given time is updated to "2020-10-16T00:00:00Z"
        And the perpetual oracles from "0xCAFECAFE1":
            | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
            | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.00          | 0.0               | 0.0               | ETH        | 1                   |
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
            | ETH/DEC19 | ETH        | USD   | my-log-normal-risk-model | default-margin-calculator | 120              | default-none | my-price-monitoring | perp-oracle        | 1e6                    | 1e6                       | -3                      | perp        | SLA        |
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
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | aux2   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
            | lpprov | ETH/DEC19 | buy  | 100000 | 1     | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2   | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 35    | 0                | TYPE_LIMIT | TIF_GTC |
            | lpprov | ETH/DEC19 | sell | 100000 | 35    | 0                | TYPE_LIMIT | TIF_GTC |
        And the market data for the market "ETH/DEC19" should be:
            | target stake | supplied stake |
            | 4315         | 100000         |
        And the opening auction period ends for market "ETH/DEC19"

    @Perpetual @twap
    Scenario: 0053-PERP-028 Internal and External TWAP calculation, auction in funding period
        Given the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
        And time is updated to "2020-10-16T00:05:00Z"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC |
        Then time is updated to "2020-10-16T00:10:00Z"

        # 1602806400 + 120s = 1602807000
        # funding period is ended with perp.funding.cue
        And the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name             | value      | time offset |
            | perp.ETH.value   | 110        | -1s         |
            | perp.funding.cue | 1602807000 | 0s          |
        And the mark price should be "10" for the market "ETH/DEC19"

        # 1 min in to the next funding period
        Given time is updated to "2020-10-16T00:11:00Z"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 11    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 11    | 1                | TYPE_LIMIT | TIF_GTC |
        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 90    | -1s         |

        # 3 min in to the next funding period
        Given time is updated to "2020-10-16T00:13:00Z"
        And the mark price should be "11" for the market "ETH/DEC19"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC |
        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 100   | -1s         |

        # 5 min in to the next funding period market moves into auction
        Given time is updated to "2020-10-16T00:15:00Z"
        And the mark price should be "10" for the market "ETH/DEC19"
        # this spot price is not counted in external twap calclation because it was broadcast during auction
        # if it did then the External spot price would be pushed to 11.625 or 11 since field type is int
        When the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 120   | -1s         |
        And the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     | reference        |
            | aux2  | ETH/DEC19 | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | trigger-auction2 |
            | aux   | ETH/DEC19 | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | trigger-auction1 |
        And the mark price should be "10" for the market "ETH/DEC19"
        Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

        ### 6 mins in Now we cancel the aux2 order, its served its purpose
        Given the parties cancel the following orders:
            | party | reference        |
            | aux   | trigger-auction1 |
            | aux2  | trigger-auction2 |
        When time is updated to "2020-10-16T00:16:00Z"
        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 110   | -1s         |
        And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

        # 7 mins in, the auction period will end
        Given time is updated to "2020-10-16T00:17:01Z"
        When the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
        Then the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 10     | 9     | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 10     | 9     | 1                | TYPE_LIMIT | TIF_GTC |

        # 8 mins in, the auction period will end
        Given time is updated to "2020-10-16T00:18:00Z"
        When the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
        And the markets are updated:
            | id        | price monitoring | linear slippage factor | quadratic slippage factor |
            | ETH/DEC19 | default-none     | 1e6                    | 1e6                       |
        Then the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 8     | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 8     | 1                | TYPE_LIMIT | TIF_GTC |
        And the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 80    | -1s         |

        # 9 mins in, the auction period will end
        Given time is updated to "2020-10-16T00:19:00Z"
        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 140   | -1s         |

        # 10 mins in, the auction period will end
        Given time is updated to "2020-10-16T00:20:00Z"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 30    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 30    | 1                | TYPE_LIMIT | TIF_GTC |
        # in theory internal TWAP = 9.625 external TWAP = 10.25, if the auction period is excluded
        # but these are type int so the decimal is truncated
        Then the product data for the market "ETH/DEC19" should be:
            | internal twap | external twap |
            | 9             | 10            |



