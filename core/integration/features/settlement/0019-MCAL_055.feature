Feature: Test funding margin for Perps market in isolated margin mode

  Background:

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | -0.05             | 0.9               | ETH        | 18                  |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
      | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.25                   | 0                         | -3                      | perp        | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 2     |

  @Perpetual
  Scenario: (0019-MCAL-055,0019-MCAL-056) check funding margin for Perps market when clumps are -0.05 and 0.9
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
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
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 1      |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 49    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 5001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC19" should be:
      | target stake | supplied stake |
      | 1100000      | 10000000       |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    # back sure we end the block so we're in a new one after opening auction
    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | USD   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 540000 | 9460000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 540000 | 9460000 |
      | party2 | USD   | ETH/DEC19 | 132000 | 9867000 |
      | party3 | USD   | ETH/DEC19 | 132000 | 9866000 |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/DEC19 | 450000      | 540000  |
      | party2 | ETH/DEC19 | 110000      | 132000  |
      | party3 | ETH/DEC19 | 110000      | 132000  |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.3           |       |
      | party2 | ETH/DEC19 | isolated margin | 0.45          |       |
      | party3 | ETH/DEC19 | isolated margin | 0.4           |       |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 700000      | 840000  | isolated margin | 0.3           | 0     |
      | party2 | ETH/DEC19 | 360000      | 432000  | isolated margin | 0.45          | 0     |
      | party3 | ETH/DEC19 | 360000      | 432000  | isolated margin | 0.4           | 0     |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3300000      | 10000000       | 3             |
    # send in external data to the perpetual market, it should not change anything and a MTM should not happen
    When the network moves ahead "1" blocks
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 1600000000000000000000 | 0s          |
      | perp.funding.cue | 1612998252             | 0s          |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/DEC19 | 900000 | 9100000 | 0            |
      | party2 | USD   | ETH/DEC19 | 450000 | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 800000 | 9198000 |              |

    # move to the block before we should MTM and check for no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 700000      | 840000  | isolated margin | 0.3           | 0     |
      | party2 | ETH/DEC19 | 360000      | 432000  | isolated margin | 0.45          | 0     |
      | party3 | ETH/DEC19 | 360000      | 432000  | isolated margin | 0.4           | 0     |
    ## Now take us past the MTM frequency time
    When the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0       | isolated margin | 0.3           | 0     |
      | party2 | ETH/DEC19 | 720000      | 864000  | isolated margin | 0.45          | 0     |
      | party3 | ETH/DEC19 | 720000      | 864000  | isolated margin | 0.4           | 0     |
    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 900000 | USD   |
      | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 540000 | USD   |
      | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 950000 | USD   |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 950000 | USD   |
    And the cumulated balance for all accounts should be worth "330000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 2001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC |
    And the mark price should be "2000" for the market "ETH/DEC19"
    #1 year has 8760 hours,so 0.002 year would be: 8760*0.002*3600 = 63072second, so next funding time (with delta_t = 0.002) would be 1612998252+63072=1613061324
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1613061324 | 0s          |

    #funding payment = f_twap - s_twap + clamp_lower_bound*s_twap =2000-1600+(-0.05*1600)=320
    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount | asset |
      | aux2   | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 640000 | USD   |
      | party2 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 320000 | USD   |
      | party3 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 320000 | USD   |
      | market | aux    | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 960000 | USD   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general | order margin |
      | party1 | USD   | ETH/DEC19 | 0       | 9100000 | 0            |
      | party2 | USD   | ETH/DEC19 | 1080000 | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 0       | 9198000 |              |

    #move to the block before the next MTM should be no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general | order margin |
      | party1 | USD   | ETH/DEC19 | 0       | 9100000 | 0            |
      | party2 | USD   | ETH/DEC19 | 1080000 | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 0       | 9198000 |              |

    ## Now take us past the MTM frequency time and things should change
    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | USD   | ETH/DEC19 | 0       | 9100000 |
      | party2 | USD   | ETH/DEC19 | 1081000 | 9549000 |
      | party3 | USD   | ETH/DEC19 | 0       | 9198000 |

And the following transfers should happen:
  | from   | to     | from account            | to account              | market id | amount | asset |
  | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 2000   | USD   |
  | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000   | USD   |
  | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000   | USD   |

And the cumulated balance for all accounts should be worth "330000000"
And the settlement account should have a balance of "0" for the market "ETH/DEC19"

