Feature: Test internal and external twap calculation

    Background:
        # epoch time is 1602806400
        Given time is updated to "2020-10-16T00:00:00Z"
        And the perpetual oracles from "0xCAFECAFE1":
            | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
            | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.00          | 0.0               | 0.0               | ETH        | 1                   |
        And the liquidity sla params named "SLA":
            | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
            | 1.0         | 0.5                          | 1                             | 1.0                    |

        And the markets:
            | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
            | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 1e6                    | 1e6                       | -3                      | perp        | default-futures |

        And the following network parameters are set:
            | name                           | value |
            | market.auction.minimumDuration | 1     |
            | limits.markets.maxPeggedOrders | 2     |
        And the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 0s    |
        And the average block duration is "1"

    @Perpetual @twap
    Scenario: 0053-PERP-027 Internal and External TWAP calculation
        Given the following network parameters are set:
            | name                                    | value |
            | network.markPriceUpdateMaximumFrequency | 120s  |
        And the parties deposit on asset's general account the following amount:
            | party  | asset | amount    |
            | party1 | USD   | 10000000  |
            | party2 | USD   | 10000000  |
            | party3 | USD   | 10000000  |
            | aux    | USD   | 100000000 |
            | aux2   | USD   | 100000000 |
            | lpprov | USD   | 100000000 |

        When the parties submit the following liquidity provision:
            | id  | party  | market id | commitment amount | fee   | lp type    |
            | lp1 | lpprov | ETH/DEC19 | 100000            | 0.001 | submission |

        # move market to continuous
        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | lpprov | ETH/DEC19 | buy  | 1000   | 5     | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | buy  | 1      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
            | lpprov | ETH/DEC19 | sell | 1000   | 15    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2   | ETH/DEC19 | buy  | 1      | 11    | 0                | TYPE_LIMIT | TIF_GTC |
            | aux    | ETH/DEC19 | sell | 1      | 11    | 0                | TYPE_LIMIT | TIF_GTC |

        And the market data for the market "ETH/DEC19" should be:
            | target stake | supplied stake |
            | 12100        | 100000         |
        Then the opening auction period ends for market "ETH/DEC19"

        Given the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
        When time is updated to "2020-10-16T00:02:00Z"
        # 1602806400 + 120s = 1602806520
        # funding period is ended with perp.funding.cue
        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name             | value      | time offset |
            | perp.ETH.value   | 110        | -1s         |
            | perp.funding.cue | 1602806520 | 0s          |

        # 1 min in to the next funding period
        Given time is updated to "2020-10-16T00:03:00Z"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 11    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 11    | 1                | TYPE_LIMIT | TIF_GTC |

        And the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 90    | -1s         |

        # 3 min in to the next funding period
        Given time is updated to "2020-10-16T00:05:00Z"
        And the mark price should be "11" for the market "ETH/DEC19"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 10    | 1                | TYPE_LIMIT | TIF_GTC |

        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 100   | -1s         |


        # 5 min in to the next funding period
        Given time is updated to "2020-10-16T00:07:00Z"
        And the mark price should be "10" for the market "ETH/DEC19"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 9     | 1                | TYPE_LIMIT | TIF_GTC |

        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 120   | -1s         |

        # 6 min in to the funding period emit spot price
        Given time is updated to "2020-10-16T00:08:00Z"
        And the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 110   | -1s         |

        # 7 mins in to the funding period
        When time is updated to "2020-10-16T00:09:00Z"

        And the mark price should be "9" for the market "ETH/DEC19"

        And the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 8     | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 8     | 1                | TYPE_LIMIT | TIF_GTC |

        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 80    | -1s         |

        # 9 min in to the next funding period
        Given time is updated to "2020-10-16T00:11:00Z"
        And the mark price should be "8" for the market "ETH/DEC19"
        When the parties place the following orders:
            | party  | market id | side | volume | price | resulting trades | type       | tif     |
            | party1 | ETH/DEC19 | buy  | 1      | 7     | 0                | TYPE_LIMIT | TIF_GTC |
            | party2 | ETH/DEC19 | sell | 1      | 7     | 1                | TYPE_LIMIT | TIF_GTC |

        Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
            | name           | value | time offset |
            | perp.ETH.value | 140   | -1s         |

        # if the funding period ended here, check the twap
        Given time is updated to "2020-10-16T00:12:00Z"

        # in theory internal TWAP = 9.3 external TWAP = 10.3
        # but these are type int so the decimal is truncated
        Then the product data for the market "ETH/DEC19" should be:
            | internal twap | external twap |
            | 10            | 10            |
