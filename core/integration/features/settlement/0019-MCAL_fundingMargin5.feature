Feature: check when settlement data precision is different/equal to the settlement asset precision, and also check decimal on the market with different market/position/asset decimal

  Background:

    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |

    And the perpetual oracles from "0xCAFECAFE1":
      | name          | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle-1 | USD | perp.ETH.value | TYPE_INTEGER | perp.funding.cue | TYPE_TIMESTAMP | 0.5 | 0.05 | 0.1 | 0.5 | ETH | 5 |
    And the perpetual oracles from "0xCAFECAFE2":
      | name          | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle-2 | USD | perp.ETH.value | TYPE_INTEGER | perp.funding.cue | TYPE_TIMESTAMP | 0.5 | 0.05 | 0.1 | 0.5 | ETH | 2 |
    And the perpetual oracles from "0xCAFECAFE3":
      | name          | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle-3 | USD | perp.ETH.value | TYPE_INTEGER | perp.funding.cue | TYPE_TIMESTAMP | 0.5 | 0.05 | 0.1 | 0.5 | ETH | 1 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
      | ETH/DEC19 | ETH | USD | default-simple-risk-model-3 | default-margin-calculator | 1 | default-none | default-none | perp-oracle-1 | 1e6 | 1e6 | -1 | perp | default-futures |
# | ETH/DEC20 | ETH | USD | default-simple-risk-model-3 | default-margin-calculator | 1 | default-none | default-none | perp-oracle-2 | 1e6 | 1e6 | -1 | perp | default-futures |
# | ETH/DEC21 | ETH | USD | default-simple-risk-model-3 | default-margin-calculator | 1 | default-none | default-none | perp-oracle-3 | 1e6 | 1e6 | -1 | perp | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 2     |

  @Perpetual
  Scenario: 0070-MKTD-018, 0070-MKTD-019, 0070-MKTD-020
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount        |
      | party1 | USD | 1000000000 |
      | party2 | USD | 100000000  |
      | party3 | USD | 100000000  |
      | aux    | USD | 300000000  |
      | aux2   | USD | 300000000  |
      | lp1    | USD | 500000000  |
      | lp2    | USD | 500000000  |
      | lp3    | USD | 500000000  |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1 | ETH/DEC19 | 1200000 | 0.001 | submission |
# | lp2 | lp2 | ETH/DEC20 | 1200000 | 0.001 | submission |
# | lp3 | lp3 | ETH/DEC21 | 1200000 | 0.001 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1 | ETH/DEC19 | 20 | 1 | buy  | BID | 50 | 1 | lp-buy-1  |
      | lp1 | ETH/DEC19 | 20 | 1 | sell | ASK | 50 | 1 | lp-sell-1 |
# | lp2 | ETH/DEC20 | 20 | 1 | buy  | BID | 50 | 1 | lp-buy-2  |
# | lp2 | ETH/DEC20 | 20 | 1 | sell | ASK | 50 | 1 | lp-sell-2 |
# | lp3 | ETH/DEC21 | 20 | 1 | buy  | BID | 50 | 1 | lp-buy-3  |
# | lp3 | ETH/DEC21 | 20 | 1 | sell | ASK | 50 | 1 | lp-sell-3 |

#place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux | ETH/DEC19 | buy  | 1 | 849  | 0 | TYPE_LIMIT | TIF_GTC |
      | aux | ETH/DEC19 | sell | 1 | 2001 | 0 | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
# When the parties place the following orders:
#   | party | market id | side | volume | price | resulting trades | type       | tif     |
#   | aux  | ETH/DEC20 | buy  | 1 | 849  | 0 | TYPE_LIMIT | TIF_GTC |
#   | aux  | ETH/DEC20 | sell | 1 | 2001 | 0 | TYPE_LIMIT | TIF_GTC |
#   | aux2 | ETH/DEC20 | buy  | 1 | 1000 | 0 | TYPE_LIMIT | TIF_GTC |
#   | aux  | ETH/DEC20 | sell | 1 | 1000 | 0 | TYPE_LIMIT | TIF_GTC |
# When the parties place the following orders:
#   | party | market id | side | volume | price | resulting trades | type       | tif     |
#   | aux   | ETH/DEC21 | buy  | 1      | 849   | 0                | TYPE_LIMIT | TIF_GTC |
#   | aux   | ETH/DEC21 | sell | 1      | 2001  | 0                | TYPE_LIMIT | TIF_GTC |
#   | aux2  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
#   | aux   | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

# And the market data for the market "ETH/DEC19" should be:
#   | target stake | supplied stake |
#   | 1100000      | 1200000        |
# And the market data for the market "ETH/DEC20" should be:
#   | target stake | supplied stake |
#   | 1100000      | 1200000        |
# And the market data for the market "ETH/DEC21" should be:
#   | target stake | supplied stake |
#   | 1100000      | 1200000        |
    Then the opening auction period ends for market "ETH/DEC19"
# Then the opening auction period ends for market "ETH/DEC20"
# Then the opening auction period ends for market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
# And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
# And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
# And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    # back sure we end the block so we're in a new one after opening auction
    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party1 | USD   | ETH/DEC19 | 120000  | 999880000 |
      | party2 | USD   | ETH/DEC19 | 132000  | 99867000  |
      | lp1    | USD   | ETH/DEC19 | 6600000 | 492200000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy | 1 | 2000 | 1 | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general    |
      | party1 | USD | ETH/DEC19 | 1441200 | 998558800 |
      | party2 | USD   | ETH/DEC19 | 132000  | 99867000   |
      | party3 | USD   | ETH/DEC19 | 132000  | 99866000   |
      | lp1 | USD | ETH/DEC19 | 6600000 | 492200000 |

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1613061324 | 0s          |

    When the network moves ahead "5" blocks

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 300000000  | 0s          |
      | perp.funding.cue | 1628766252 | 0s          |
# #1628766252 is half year after the first oracel time

    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount | asset |
      | aux2   | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 74700  | USD   |
      | party2 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 74700  | USD   |
      | party3 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 74700  | USD   |
      | market | aux    | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 74700  | USD   |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 149400 | USD   |

    When the network moves ahead "5" blocks

    And the markets are updated:
      | id        | data source config | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | perp-oracle-2      | 1e6                    | 1e6                       |

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1628766260 | 0s          |

    When the network moves ahead "5" blocks

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.ETH.value   | 350000000  | 0s          |
      | perp.funding.cue | 1728766260 | 0s          |










