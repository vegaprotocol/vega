Feature: check when settlement data precision is different/equal to the settlement asset precision

  Background:

    And the following assets are registered:
      | id  | decimal places |
      | USD | 2              |
    And the perpetual oracles from "0xCAFECAFE1":
      | name          | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle-1 | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.15              | ETH        | 5                   |
    And the perpetual oracles from "0xCAFECAFE2":
      | name          | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle-2 | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.15              | ETH        | 2                   |
    And the perpetual oracles from "0xCAFECAFE3":
      | name          | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle-3 | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.15              | ETH        | 1                   |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | position decimal places | market type | sla params      |
      | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle-1      | 1e6                    | 1e6                       | -1                      | perp        | default-futures |

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
      | party1 | USD | 100000000000000 |
      | party2 | USD   | 100000000     |
      | party3 | USD   | 100000000     |
      | aux    | USD   | 1000000       |
      | aux2   | USD   | 1000000       |
      | lpprov | USD   | 5000000       |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 1200000           | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 1200000           | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lpprov | ETH/DEC19 | 20        | 1                    | buy  | BID              | 50     | 1      | lp-buy    |
      | lpprov | ETH/DEC19 | 20        | 1                    | sell | ASK              | 50     | 1      | lp-sell   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux | ETH/DEC19 | buy  | 1 | 849  | 0 | TYPE_LIMIT | TIF_GTC |
      | aux | ETH/DEC19 | sell | 1 | 2001 | 0 | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC19" should be:
      | target stake | supplied stake |
      | 1100000      | 1200000        |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    And the orders should have the following status:
      | party  | reference | status           |
      | lpprov | lp-sell   | STATUS_CANCELLED |
      | lpprov | lp-buy    | STATUS_CANCELLED |

    # back sure we end the block so we're in a new one after opening auction
    When the network moves ahead "1" blocks

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general      |
      | party1 | USD | ETH/DEC19 | 120000 | 99999999880000 |
      | party2 | USD   | ETH/DEC19 | 132000 | 99867000     |
      | lpprov | USD | ETH/DEC19 | 0 | 3800000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin         | general        |
      | party1 | USD   | ETH/DEC19 | 13200000240000 | 86799999760000 |

    And the orders should have the following status:
      | party  | reference | status           |
      | lpprov | lp-sell   | STATUS_CANCELLED |
      | lpprov | lp-buy    | STATUS_CANCELLED |



