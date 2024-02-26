Feature: Test mark to market settlement with periodicity, takes the first scenario from mark_to_market_settlement_neg_pdp

  Background:

    Given the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 18                  |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 1e6                    | 1e6                         | -3                       | perp        | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 2     |

  @Perpetual @FundingLoss
  Scenario: (0053-PERP-005) Mark to market settlement works correctly with a predefined frequency irrespective of the behaviour of any of the oracles specified for the market.
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 1      |
 
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
      | party1 | ETH   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | ETH   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 9866000 |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |


    # send in external data to the perpetual market, funding payment is triggered
    When the network moves ahead "1" blocks
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                   | time offset |
      | perp.ETH.value   | 2100000000000000000000  | 0s         |
      | perp.funding.cue | 1511924180              | 0s          |
    Then the transfers of following types should NOT happen:
      | type                   |
      | TRANSFER_TYPE_MTM_WIN  |
      | TRANSFER_TYPE_MTM_LOSS |

    # move to the block before we should MTM and check for no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 9866000 |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |

    ## Now take us past the MTM frequency time
    When the network moves ahead "1" blocks
    Then the transfers of following types should happen:
      | type                   |
      | TRANSFER_TYPE_MTM_WIN  |
      | TRANSFER_TYPE_MTM_LOSS |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 7682400 | 1317600 |
      | party3 | ETH   | ETH/DEC19 | 2605200 | 7392800 |
      | party2 | ETH   | ETH/DEC19 | 2605200 | 8393800 |
    And the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount  | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | ETH   |
    And the cumulated balance for all accounts should be worth "330000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the network moves ahead "1" blocks
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 2001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | -1000000       | 0            |
      | party2 | 1      | 1000000        | 0            |
      | party3 | 1      | 0              | 0            |

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1575072009 | 0s          |
    # funding payment is -815
    Then the following funding period events should be emitted:
      | start      | end        | internal twap | external twap | funding payment |
      | 1511924180 | 1575072009 | 1285          | 2100          | -815            |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 7370000 | 0       |
      | party2 | ETH   | ETH/DEC19 | 2605200 | 9208800 |
      | party3 | ETH   | ETH/DEC19 | 2605200 | 8207800 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | -1000000       | -1630000     |
      | party2 | 1      | 1000000        | 815000       |
      | party3 | 1      | 0              | 815000       |

    # move to the block before the next MTM should be no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 7370000 | 0       |
      | party2 | ETH   | ETH/DEC19 | 2605200 | 9208800 |
      | party3 | ETH   | ETH/DEC19 | 2605200 | 8207800 |

    ## Now take us past the MTM frequency time and things should change
    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 7368000 | 0       |
      | party2 | ETH   | ETH/DEC19 | 2606200 | 9208800 |
      | party3 | ETH   | ETH/DEC19 | 2606200 | 8207800 |
    And the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount  | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1630000 | ETH   |
    And the cumulated balance for all accounts should be worth "330000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

  @Perpetual @PerpMargin
  Scenario: A party that never holds a position should end with margin levels at zero
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 10000000  |
      | party4 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 1      |

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
      | party1 | ETH   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | ETH   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1999  | 0                | TYPE_LIMIT | TIF_GTC | p3-buy-1  |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 9868000 |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |


    # send in external data to the perpetual market, it should not change anything and a MTM should not happen
    When the network moves ahead "1" blocks
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 2100000000000000000000 | 0s          |
      | perp.funding.cue | 1511924180             | 0s          |

    # move to the block before we should MTM and check for no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 1440000 | 8560000 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 9868000 |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |
    And the cumulated balance for all accounts should be worth "340000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 2001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |
      | party3 | 0      | 0              | 0            |
    # make sure the margin levels for party3 have not changed
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC | p4-buy-1  |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party4 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 1440000 | 8560000 |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 9868000 |
      | party4 | ETH   | ETH/DEC19 | 132000  | 9865999 |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | -1002000       | 0            |
      | party2 | 1      | 1001000        | 0            |
      | party3 | 0      | 0              | 0            |
      | party4 | 1      | 0              | 0            |
    When the parties cancel the following orders:
      | party  | reference | error |
      | party3 | p3-buy-1  |       |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general  |
      | party1 | ETH   | ETH/DEC19 | 7680240 | 1317760  |
      | party2 | ETH   | ETH/DEC19 | 266532  | 10733468 |
      | party3 | ETH   | ETH/DEC19 | 0       | 10000000 |
      | party4 | ETH   | ETH/DEC19 | 266532  | 9731467  |
    And the cumulated balance for all accounts should be worth "340000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

  @Perpetual @PerpMargin
  Scenario: A party that never held a position should simply have its orders closed, but never see their margin get confiscated
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 1000000   |
      | party4 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 1      |

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
      | party1 | ETH   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | ETH   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 1      | 1999  | 0                | TYPE_LIMIT | TIF_GTC | p3-buy-1  |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 868000  |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |


    # send in external data to the perpetual market, it should not change anything and a MTM should not happen
    When the network moves ahead "1" blocks
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 2100000000000000000000 | 0s          |
      | perp.funding.cue | 1511924180             | 0s          |

    # move to the block before we should MTM and check for no changes
    When the network moves ahead "3" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 1440000 | 8560000 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 868000  |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |
    And the cumulated balance for all accounts should be worth "331000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 2001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -1     | 0              | 0            |
      | party2 | 1      | 0              | 0            |
      | party3 | 0      | 0              | 0            |
    # make sure the margin levels for party3 have not changed
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC19 | buy  | 1      | 2001  | 1                | TYPE_LIMIT | TIF_GTC | p4-buy-1  |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party4 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 1440000 | 8560000 |
      | party2 | ETH   | ETH/DEC19 | 132000  | 9867000 |
      | party3 | ETH   | ETH/DEC19 | 132000  | 868000  |
      | party4 | ETH   | ETH/DEC19 | 132000  | 9865999 |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -2     | -1002000       | 0            |
      | party2 | 1      | 1001000        | 0            |
      | party3 | 0      | 0              | 0            |
      | party4 | 1      | 0              | 0            |
    # let's get party3 to run out of margin
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 2      | 1999  | 0                | TYPE_LIMIT | TIF_GTC | p3-buy-2  |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 660330      | 726363 | 792396  | 924462  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general  |
      | party1 | ETH   | ETH/DEC19 | 7680240 | 1317760  |
      | party2 | ETH   | ETH/DEC19 | 266532  | 10733468 |
      | party3 | ETH   | ETH/DEC19 | 792396  | 207604   |
      | party4 | ETH   | ETH/DEC19 | 266532  | 9731467  |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | sell | 1      | 4001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 4001  | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC19 | 0           | 0      | 0       | 0       |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 3680240 | 1317760 |
      | party2 | ETH   | ETH/DEC19 | 5270532 | 7729468 |
      | party3 | ETH   | ETH/DEC19 | 0       | 1000000 |
      | party4 | ETH   | ETH/DEC19 | 5270532 | 6727467 |

  @Perpetual @PerpMargin @PerpMarginBug
  Scenario: Verify margins are adjusted for funding payments as expected
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 10000000   |
      | party2 | ETH   | 10000000   |
      | party3 | ETH   | 1000000    |
      | party4 | ETH   | 10000000   |
      | aux    | ETH   | 100000000  |
      | aux2   | ETH   | 100000000  |
      | lpprov | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 100000000         | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 100000000         | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 1      |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 49    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 5001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC19" should be:
      | target stake | supplied stake |
      | 1100000      | 100000000      |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    # back sure we end the block so we're in a new one after opening auction
    When the network moves ahead "1" blocks
    # set pegs 1001 and 999 to keep margins consistent
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | ETH   | ETH/DEC19 | 132000 | 9867000 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 100000      | 110000 | 120000  | 140000  |
      | party2 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |
      | party3 | ETH/DEC19 | 100000      | 110000 | 120000  | 140000  |
      | party4 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    # funding payment -> external TWAP is based on 1100, short parties are losing, their margin increases more
    When the network moves ahead "2" blocks
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 1100000000000000000000 | -1s         |
      | perp.funding.cue | 1511924180             | -1s         |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 101000      | 111100 | 121200  | 141400  |
      | party2 | ETH/DEC19 | 111000      | 122100 | 133200  | 155400  |
      | party3 | ETH/DEC19 | 100000      | 110000 | 120000  | 140000  |
      | party4 | ETH/DEC19 | 110000      | 121000 | 132000  | 154000  |

    # Repeat the same operations, create new internal TWAP data point at the same price levels
    When the network moves ahead "1" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    # Margins obviously increase as the position increased
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/DEC19 | 201000      | 221100 | 241200  | 281400  |
      | party2 | ETH/DEC19 | 221000      | 243100 | 265200  | 309400  |
      | party3 | ETH/DEC19 | 200000      | 220000 | 240000  | 280000  |
      | party4 | ETH/DEC19 | 220000      | 242000 | 264000  | 308000  |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    # funding payment -> external TWAP is based on 900, long parties are losing, their margin increases more
    When the network moves ahead "1" blocks
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                 | time offset |
      | perp.ETH.value   | 900000000000000000000 | -1s         |
      | perp.funding.cue | 1511924181            | -1s         |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party2 | ETH/DEC19 | 222000      | 244200 | 266400  | 310800  |
      | party1 | ETH/DEC19 | 202000      | 222200 | 242400  | 282800  |
      | party3 | ETH/DEC19 | 200000      | 220000 | 240000  | 280000  |
      | party4 | ETH/DEC19 | 220000      | 242000 | 264000  | 308000  |
