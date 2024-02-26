Feature: Test funding payment triggering closeout for Perps market

  Background:

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.9               | ETH        | 18                  |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
      | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 1e6                    | 1e6                       | -3                      | perp        | default-futures |

    And the initial insurance pool balance is "100" for all the markets
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 2     |

  @Perpetual @Liquidation
  Scenario: (0053-PERP-024) Funding payment triggering closeout but no loss soccialization
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | USD   | 10000000  |
      | party2 | USD   | 10000000  |
      | party3 | USD   | 10000000  |
      | aux    | USD   | 100000000 |
      | aux2   | USD   | 2391000   |
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

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |

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
    When the network moves ahead "4" blocks

    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount  | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | USD   |
      | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | USD   |
      | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000000 | USD   |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000000 | USD   |
    # And the cumulated balance for all accounts should be worth "330010000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 2001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/DEC19 | 6402000     | 7682400 |
      | party2 | ETH/DEC19 | 2171000     | 2605200 |
      | aux2   | ETH/DEC19 | 2391000     | 2869200 |
    And the mark price should be "2000" for the market "ETH/DEC19"
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | -1     | -1000000       |  96000       |
      | aux2   | 1      | 1000000        | -96000       |
      | party1 | -2     | -1000000       | 0            |
      | party2 | 1      | 1000000        | 0            |
      | party3 | 1      | 0              | 0            |
      | lpprov | 0      | 0              | 0            |
    ## allow close-outs to happen
    When the network moves ahead "1" blocks
    #1 year has 8760 hours,so 0.002 year would be: 8760*0.002*3600 = 63072second, so next funding time (with delta_t = 0.002) would be 1612998252+63072=1613061324
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1613061324 | 0s          |

    #funding payment = f_twap - s_twap + clamp_lower_bound*s_twap =2000-1600+(0.1*1600)=560
    Then the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount  | asset |
      | aux2   | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1120000 | USD   |
      | party2 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 560000  | USD   |
      | party3 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 560000  | USD   |
      | market | aux    | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1120000 | USD   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1120000 | USD   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | USD   | ETH/DEC19 | 7682400 | 2629600 |
      | party3 | USD   | ETH/DEC19 | 2605200 | 6736800 |
      | party2 | USD   | ETH/DEC19 | 2605200 | 7737800 |
      | aux2   | USD   | ETH/DEC19 | 0       | 0       |

    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | aux    | -1     | -1000000       | 1216000      |
      | aux2   | 0      | 0              | -2388999     |
      | party1 | -2     | -1000000       | 1120000      |
      | party2 | 1      | 1000000        | -560000      |
      | party3 | 1      | 0              | -560000      |
      | lpprov | 0      | 0              | 0            |

    And the insurance pool balance should be "2173099" for the market "ETH/DEC19"



