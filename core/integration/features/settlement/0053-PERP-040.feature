Feature: Assert that the scaling factor is applied before the funding cap is applied (0053-PERP-040).

  @Perpetual
  Scenario: Make sure that the scaling is handled first and the bounds do not touch the result

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | funding rate scaling factor | funding rate lower bound | funding rate upper bound |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 1.0                   | 0.0           | 0.1               | 0.9               | ETH        | 18                  | 10.0                        | -6.0                     | 6.0                      |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
      | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.25                   | 0                         | -3                      | perp        | default-futures |
    And the initial insurance pool balance is "200" for all the markets
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 2     |

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | aux    | USD   | 100000000 |
      | aux2   | USD   | 100000000 |
      | lpprov | USD   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
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

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # back sure we end the block so we're in a new one after opening auction
    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 1      | 1200  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 1      | 1200  | 1                | TYPE_LIMIT | TIF_GTC |

    # send in external data to the perpetual market, it should not change anything and a MTM should not happen
    When the network moves ahead "1" blocks
    Then system unix time is "1575072004"

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 3000000000000000000000 | 0s          |
      | perp.funding.cue | 1575072004             | 0s          |
    When the network moves ahead "6" blocks
    
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 3000000000000000000000 | 0s          |
      | perp.funding.cue | 1575072012             | 0s          |
    # funding payments = s_twap * delta_t * interest_rate * fund rate scaling * dp = (3000-1200)*1*10.0*1000 = 18,000,000
    # funding rate will be funding payment / s_twap = 18,000,000/(3,000*dp) = 6
    # So having a lower/upper bound of -6/+6 will not change the result
    And the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount  | asset |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 18000000| USD   |



  @Perpetual
  Scenario: Make sure that the scaling is handled first and the bounds can change the final result

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals | funding rate scaling factor | funding rate lower bound | funding rate upper bound |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 1.0                   | 0.0           | 0.1               | 0.9               | ETH        | 18                  | 10.0                        | -5.0                     | 5.0                      |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
      | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.25                   | 0                         | -3                      | perp        | default-futures |
    And the initial insurance pool balance is "200" for all the markets
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 2     |

    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | aux    | USD   | 100000000 |
      | aux2   | USD   | 100000000 |
      | lpprov | USD   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
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

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # back sure we end the block so we're in a new one after opening auction
    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 1      | 1200  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 1      | 1200  | 1                | TYPE_LIMIT | TIF_GTC |

    # send in external data to the perpetual market, it should not change anything and a MTM should not happen
    When the network moves ahead "1" blocks
    Then system unix time is "1575072004"
    
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 3000000000000000000000 | 0s          |
      | perp.funding.cue | 1575072004             | 0s          |
    When the network moves ahead "6" blocks

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 3000000000000000000000 | 0s          |
      | perp.funding.cue | 1575072012             | 0s          |

    # funding payments = s_twap * delta_t * interest_rate * fund rate scaling * dp = (3000-1200)*1*10.0*1000 = 18,000,000
    # funding rate will be funding payment / s_twap = 18,000,000/(3,000*dp) = 6
    # So having a lower/upper bound of -5/+5 if applied after the scaling will reduce the payment to 15,000,000
    # If the bounds are incorrectly handled first the result will be the same as the run above, 18,000,000
    And the following transfers should happen:
      | from   | to     | from account            | to account          | market id | amount   | asset |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN | ETH/DEC19 | 15000000 | USD   |

