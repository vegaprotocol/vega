Feature: Simple test creating a perpetual market.

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
      | USD | 0              |
    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | source weights | source staleness tolerance |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0                     | 0             | 0                 | 0                 | ETH        | 18                  | 1,0,0,0        | 100s,0s,0s,0s              |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.01             | 10s         | 5              |  

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.1                    | 0                         | 5              | 5                       | perp        | SLA        |
    And the following network parameters are set:
      | name                                             | value |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | market.auction.minimumDuration                   | 1     |
      | market.fee.factors.infrastructureFee             | 0.001 |
      | market.fee.factors.makerFee                      | 0.004 |
      | market.value.windowLength                        | 60s   |
      | market.liquidity.bondPenaltyParameter            | 0.1   |
      | validators.epoch.length                          | 5s    |
      | limits.markets.maxPeggedOrders                   | 2     |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |

    And the average block duration is "1"

    # All parties have 1,000,000.000,000,000,000,000,000
    # Add as many parties as needed here
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount                     |
      | lpprov  | ETH   | 10000000000000000000000000 |
      | trader1 | ETH   | 10000000000000000000000000 |
      | trader2 | ETH   | 10000000000000000000000000 |
      | trader3 | ETH   | 10000000000000000000000000 |
      | trader4 | ETH   | 10000000000000000000000000 |
      | trader5 | ETH   | 10000000000000000000000000 |


  @Perpetual
  Scenario: 001 Create a new perp market and leave opening auction in the same way the system tests do
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 1      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 1      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |

    # example of how to use the oracle
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1511924180 | -100s       |
      | perp.ETH.value   | 975        | -2s         |
      | perp.ETH.value   | 977        | -1s         |


  @Perpetual
  Scenario: 002 Create a new perp market and leave opening auction, then terminate the market through governance
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 1      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 1      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |


    # example of how to use the oracle
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1511924180 | -100s       |
      | perp.ETH.value   | 975        | -2s         |
      | perp.ETH.value   | 977        | -1s         |

    When the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC19 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 976              |
    Then the market state should be "STATE_CLOSED" for the market "ETH/DEC19"

  @PerpetualCancel
  Scenario: 003 Cancel a perps market in opening auction
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 1      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 1      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                 | auction trigger         |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |
    And the parties should have the following account balances:
      | party   | asset | market id | margin      | general                   |
      | trader1 | ETH   | ETH/DEC19 | 60659552186 | 9999999999999939340447814 |


    # example of how to use the oracle
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name           | value | time offset |
      | perp.ETH.value | 975   | -2s         |
      | perp.ETH.value | 977   | -1s         |

    And the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | sell | 1      | 951   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |
    When the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC19 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 976              |
    Then the market state should be "STATE_CLOSED" for the market "ETH/DEC19"

  @PerpetualCancel
  Scenario: 003 Cancel a perps market in opening auction
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size        | minimum visible size | side | pegged reference | volume           | offset | reference   |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | buy  | BID              | 4000000000000000 | 1      | lp-ice-buy  |
      | lpprov | ETH/DEC19 | 4000000000000000 | 3905000000000000     | sell | ASK              | 4000000000000000 | 1      | lp-ice-sell |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                 | auction trigger         |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |
    And the parties should have the following account balances:
      | party   | asset | market id | margin      | general                   |
      | trader1 | ETH   | ETH/DEC19 | 60659552186 | 9999999999999939340447814 |


    # example of how to use the oracle
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name           | value | time offset |
      | perp.ETH.value | 975   | -2s         |
      | perp.ETH.value | 977   | -1s         |

    And the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC19 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 976              |
    Then the market state should be "STATE_CANCELLED" for the market "ETH/DEC19"

  @Perpetual
  Scenario: 004 Create a new perp market and run through funding periods with: no oracle data -> no payment, 1 oracle observation & no trading -> payment based internal TWAP=auction uncrossing price, multiple internal and external observations => funding payment calculated as expected.
    Given the following network parameters are set:
      | name                                          | value |
      | network.internalCompositePriceUpdateFrequency | 0s    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 400000000 | 100                  | buy  | BID              | 8000000000 | 1      |
      | lpprov | ETH/DEC19 | 400000000 | 100                  | sell | ASK              | 8000000000 | 1      |
    And the parties place the following orders:
      | party   | market id | side | volume | price | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 1001  | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      |  951  | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC19"
    Then system unix time is "1575072002"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1             |
    And the following funding period events should be emitted:
      | start      | end        | internal twap    | external twap    |
      | 1575072002 |            | 9760000000000000 |                  |

    # perps payment doesn't happen in the absence of oracle data
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1575072003 |  0s         |
    And the following funding period events should be emitted:
      | start      | end         | internal twap    | external twap    |
      | 1575072002 | 1575072003  | 9760000000000000 |                  |
      | 1575072003 |             | 9760000000000000 |                  |
    Then the transfers of following types should NOT happen:
      | type                                  |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN  |

    # perps payment happens in absence of trades after the opening auction (opening auction sets mark price so that's already one internal observation) 
    When the network moves ahead "4" blocks
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value            | time offset |
      | perp.ETH.value   | 9770000000000000 | -1s         |
      | perp.funding.cue | 1575072007       |  0s         |
    Then system unix time is "1575072006"
    # funding payment = 976 - 977 = -1
    # TODO: is 13 decimal places fine, it's asset - market so seems like internal precision, do we ever use that externally?? It's fine on API but would be nice to understand   
    And the following funding period events should be emitted:
      | start      | end         | internal twap    | external twap    | funding payment | funding rate        |
      | 1575072003 | 1575072007  | 9760000000000000 | 9770000000000000 | -10000000000000 | -0.0010235414534289 |
      | 1575072007 |             | 9760000000000000 | 9770000000000000 |                 |                     | 
    And the following transfers should happen:
      | type                                  | from    | to      | from account             | to account              | market id | amount    | asset |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS | trader2 | market  | ACCOUNT_TYPE_MARGIN      | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 100000000 | ETH   |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN  | market  | trader1 | ACCOUNT_TYPE_SETTLEMENT  | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 100000000 | ETH   |

    # perps payment calculated correctly with multiple internal and external observations
    When the network moves ahead "2" blocks

    Then the mark price should be "976" for the market "ETH/DEC19"

    And the parties place the following orders with ticks:
      | party   | market id | side | volume | price | type       | tif     | resulting trades |
      | trader3 | ETH/DEC19 | buy  | 1      |  980  | TYPE_LIMIT | TIF_GTC | 0                |
      | trader4 | ETH/DEC19 | sell | 1      |  980  | TYPE_LIMIT | TIF_FOK | 1                |

    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             |
      | 980        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |
    And system unix time is "1575072008"

    When the network moves ahead "4" blocks
    And the parties place the following orders with ticks:
      | party   | market id | side | volume | price | type       | tif     | resulting trades |
      | trader3 | ETH/DEC19 | buy  | 1      |  989  | TYPE_LIMIT | TIF_GTC | 0                |
      | trader4 | ETH/DEC19 | sell | 1      |  989  | TYPE_LIMIT | TIF_FOK | 1                |
    
    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader3 | 989   | 1    | trader4 |
    
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             |
      | 989        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |
    And system unix time is "1575072012"
    
    When the network moves ahead "1" blocks
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value            | time offset |
      | perp.ETH.value   | 9760000000000000 | -5s         |
      # resubmitting same value at a different time should have no effect
      | perp.ETH.value   | 9760000000000000 | -2s         |
      | perp.ETH.value   | 9720000000000000 | -1s         |
      | perp.funding.cue | 1575072014       |  0s         |
    Then system unix time is "1575072013"
    # internal TWAP = (976*1+980*4+989*2)/7=982
    # external TWAP = (977*1+976*4+972*2)/7=975
    # funding payment = 7
    Then debug funding period events

    And the following funding period events should be emitted:
      | start      | end        | internal twap    | external twap    | funding payment | funding rate       |
      | 1575072007 | 1575072014 | 9820000000000000 | 9750000000000000 | 70000000000000  | 0.0071794871794872 |
      | 1575072014 |            | 9890000000000000 | 9720000000000000 |                 |                    |
    # payments for trader3 and trader4 should be twice those of trader1 and trader2
    And the following transfers should happen:
      | type                                  | from    | to      | from account             | to account              | market id |  amount    | asset |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS | trader1 | market  | ACCOUNT_TYPE_MARGIN      | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |  700000000 | ETH   |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN  | market  | trader2 | ACCOUNT_TYPE_SETTLEMENT  | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 |  700000000 | ETH   |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS | trader3 | market  | ACCOUNT_TYPE_MARGIN      | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1400000000 | ETH   |
      | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN  | market  | trader4 | ACCOUNT_TYPE_SETTLEMENT  | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1400000000 | ETH   |
    
