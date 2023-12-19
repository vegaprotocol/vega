Feature: Test funding margin for Perps market under isolated margin mode

  Background:

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.9               | ETH        | 18                  |
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
  Scenario: (0019-MCAL-053) check funding margin for Perps market when clumps are 0.1 and 0.9 in isolated margin mode
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
      | lp2 | party1 | ETH/DEC19 | 1000              | 0.001 | submission |
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
      | party1 | USD   | ETH/DEC19 | 120000 | 9879000 |
      | party2 | USD   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party1 | ETH/DEC19 | sell | 2      | 2000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell-0 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 660000 | 9339000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 660000 | 9339000 |
      | party2 | USD   | ETH/DEC19 | 132000 | 9867000 |
      | party3 | USD   | ETH/DEC19 | 132000 | 9866000 |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.3           |       |
      | party2 | ETH/DEC19 | isolated margin | 0.45          |       |
      | party3 | ETH/DEC19 | isolated margin | 0.4           |       |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | error | reference   |
      | party1 | ETH/DEC19 | sell | 1      | 3000  | 0                | TYPE_LIMIT | TIF_GTC |       | party1-sell |
    And the network moves ahead "2" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial | margin mode     | margin factor | order   |
      | party1 | ETH/DEC19 | 700000      | 840000  | isolated margin | 0.3           | 1500000 |
      | party2 | ETH/DEC19 | 360000      | 432000  | isolated margin | 0.45          | 0       |
      | party3 | ETH/DEC19 | 360000      | 432000  | isolated margin | 0.4           | 0       |

    When the network moves ahead "1" blocks

    #order margin: 2000*1*0.3+3000*1*0.3=1500
    #position margin: 1000*1*0.3+2000*1*0.3=900
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/DEC19 | 900000 | 7599000 | 1500000      | 1000 |
      | party2 | USD   | ETH/DEC19 | 450000 | 9549000 |              |      |
      | party3 | USD   | ETH/DEC19 | 800000 | 9198000 |              |      |
    And the mark price should be "1000" for the market "ETH/DEC19"

    # send in external data to the perpetual market, it should not change anything and a MTM should not happen
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 1600000000000000000000 | 0s          |
      | perp.funding.cue | 1612998252             | 0s          |

    And the orders should have the following status:
      | party  | reference   | status        |
      | party1 | party1-sell | STATUS_ACTIVE |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/DEC19 | 900000 | 7599000 | 1500000      |
      | party2 | USD   | ETH/DEC19 | 450000 | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 800000 | 9198000 |              |

    # move to the block before we should MTM and check for no changes
    When the network moves ahead "3" blocks
    And the mark price should be "2000" for the market "ETH/DEC19"
    And the orders should have the following status:
      | party  | reference     | status        |
      | party1 | party1-sell-0 | STATUS_FILLED |
      | party1 | party1-sell   | STATUS_ACTIVE |

    #MTM: (2000-1500)*2*0.3=300 (1500 is entry price for party1's position)
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general | order margin |
      | party1 | USD   | ETH/DEC19 | 600000  | 7599000 | 900000       |
      | party2 | USD   | ETH/DEC19 | 1400500 | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 800000  | 9198000 |              |
    # Now take us past the MTM frequency time
    #after MTM, party1's margin account is lower than maintenance level, so position is closeout, however, one of the order is used in closed out, so party1 still has position 1
    When the network moves ahead "1" blocks

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial | margin mode     | margin factor | order  |
      | party1 | ETH/DEC19 | 700000      | 840000  | isolated margin | 0.3           | 900000 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general | order margin |
      | party1 | USD   | ETH/DEC19 | 600000  | 7599000 | 900000       |
      | party2 | USD   | ETH/DEC19 | 1400500 | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 800000  | 9198000 |              |

    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 900000 | USD   |
      | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 540000 | USD   |
      | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 950500 | USD   |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 950500 | USD   |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | -901000      |
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

    #funding payment = f_twap - s_twap + clamp_lower_bound*s_twap =2000-1600+(0.1*1600)=560
    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount  | asset |
      | aux2   | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1120000 | USD   |
      | party2 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 560000  | USD   |
      | party3 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 560000  | USD   |
      | market | aux    | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1120000 | USD   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 560000  | USD   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general | order margin |
      | party1 | USD   | ETH/DEC19 | 1160000 | 7599000 | 900000       |
      | party2 | USD   | ETH/DEC19 | 840500  | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 0       | 9198000 |              |

    #move to the block before the next MTM should be no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general | order margin |
      | party1 | USD   | ETH/DEC19 | 1160000 | 7599000 | 900000       |
      | party2 | USD   | ETH/DEC19 | 840500  | 9549000 |              |
      | party3 | USD   | ETH/DEC19 | 0       | 9198000 |              |

    ## Now take us past the MTM frequency time and things should change
    When the network moves ahead "5" blocks
    And the mark price should be "2001" for the market "ETH/DEC19"
    # Then the parties should have the following account balances:
    #   | party  | asset | market id | margin  | general | order margin |
    #   | party1 | USD   | ETH/DEC19 | 1160000 | 7599000 | 900000       |
    #   | party2 | USD   | ETH/DEC19 | 840500  | 9549000 |              |
    #   | party3 | USD   | ETH/DEC19 | 0       | 9198000 |              |

    # And the following transfers should happen:
    #   | from   | to     | from account        | to account              | market id | amount | asset |
      # | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000   | USD   |
      # | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 2000   | USD   |
#   | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000   | USD   |
#   | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000   | USD   |
#   | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000   | USD   |

# # And the cumulated balance for all accounts should be worth "330000000"
# # And the settlement account should have a balance of "0" for the market "ETH/DEC19"

