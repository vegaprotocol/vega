Feature: check the insurance pool getting shared equally between all markets with the same settlement asset + the on-chain treasury for the asset.

  Background:

    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    #calculated risk factor long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999999  | 300               |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE19" named "ethDec19Oracle":
      | property         | type         | binding         | decimals |
      | prices.ETH.value | TYPE_INTEGER | settlement data | 0        |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE19" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE20" named "ethDec20Oracle":
      | property         | type         | binding         | decimals |
      | prices.ETH.value | TYPE_INTEGER | settlement data | 0        |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE20" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE21" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE21" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |


    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | ethDec19Oracle     | 0.74667                | 0                         | default-futures |
      | ETH/DEC20 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | ethDec20Oracle     | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC21 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | ethDec21Oracle     | 1e6                    | 1e6                       | default-futures |


    And the following network parameters are set:
      | name                                         | value |
      | market.auction.minimumDuration               | 1     |
      | network.markPriceUpdateMaximumFrequency      | 0s    |
      | market.liquidity.successorLaunchWindowLength | 1s    |
      | limits.markets.maxPeggedOrders               | 4     |

  @Liquidation
  Scenario: using lognormal risk model, set "designatedLoser" closeout while the position of "designatedLoser" is not fully covered by orders on the order book; and check the funding of treasury. 0012-POSR-002, 0012-POSR-005, 0013-ACCT-001, 0013-ACCT-022

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | USD   | 1000000000000 |
      | buySideProvider  | USD   | 1000000000000 |
      | designatedLoser  | USD   | 21981         |
      | aux              | USD   | 1000000000000 |
      | aux2             | USD   | 1000000000000 |
      | lpprov           | USD   | 1000000000000 |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp2 | lpprov | ETH/DEC20 | 9000              | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 225       | 18                   | buy  | BID              | 225    | 100    |
      | lpprov | ETH/DEC19 | 36        | 18                   | sell | ASK              | 36     | 100    |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC20 | 225       | 18                   | buy  | BID              | 225    | 100    |
      | lpprov | ETH/DEC20 | 36        | 18                   | sell | ASK              | 36     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-1-19  |
      | aux   | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |           |
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # insurance pool generation - setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | 731          | 9000           | 1             |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf=150*10*1*0.4878731=731

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | volume | price |
      | buy  | 10     | 1     |
      | sell | 10     | 2000  |

    # insurance pool generation - trade
    When the parties place the following orders with ticks:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC19 | 0      | 0       |

    And the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC19 | 0           | 0      | 0       | 0       |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 1     | 0      |
      | buy  | 140   | 0      |

    When the parties place the following orders with ticks:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    # insurance pool generation - set new mark price (and trigger closeout)
    #When the parties place the following orders with ticks:
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the network moves ahead "1" blocks

    Then debug trades
    Then the following trades should be executed:
      | buyer           | price | size | seller           |
      | buySideProvider | 140   | 1    | sellSideProvider |
      | network         | 0     | 290  | designatedLoser  |
      | buySideProvider | 140   | 1    | network          |
      | lpprov          | 40    | 225  | network          |
      | aux             | 1     | 10   | network          |
      | buySideProvider | 120   | 54   | network          |

    Then the following network trades should be executed:
      | party           | aggressor side | volume |
      | buySideProvider | buy            | 1      |
      | buySideProvider | sell           | 1      |
      | lpprov          | sell           | 225    |
      | aux             | sell           | 10     |
      | designatedLoser | buy            | 290    |

    # check margin levels
    Then the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | designatedLoser | ETH/DEC19 | 0           | 0      | 0       | 0       |
    # checking margins
    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | designatedLoser | USD   | ETH/DEC19 | 0      | 0       |


    # then we make sure the insurance pool collected the funds (however they get later spent on MTM payment to closeout-facilitating party)
    Then the following transfers should happen:
      | from            | to              | from account            | to account                       | market id | amount | asset |
      | designatedLoser | market          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC19 | 0      | USD   |
      | designatedLoser | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC19 | 4350   | USD   |
      | designatedLoser |                 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC19 | 0      | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC19 | 0      | USD   |
      | designatedLoser | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_INSURANCE           | ETH/DEC19 | 17631  | USD   |
      | market          | market          | ACCOUNT_TYPE_INSURANCE  | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC19 | 16716  | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 8      | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 0      | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 8      | USD   |
      | market          | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 0      | USD   |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general      |
      | buySideProvider  | USD   | ETH/DEC19 | 24057  | 999999975387 |
      | sellSideProvider | USD   | ETH/DEC19 | 83564  | 999999919336 |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2  | ETH/DEC19 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux   | ETH/DEC19 | sell | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 120        | TRADING_MODE_CONTINUOUS | 170949       | 9000           | 292           |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    And the parties should have the following account balances:
      | party            | asset | market id | margin | general      |
      | buySideProvider  | USD   | ETH/DEC19 | 22937  | 999999975387 |
      | sellSideProvider | USD   | ETH/DEC19 | 64666  | 999999944054 |

    Then the following transfers should happen:
      | from            | to               | from account            | to account              | market id | amount | asset |
      | buySideProvider | market           | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1120   | USD   |
      | market          | sellSideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 5820   | USD   |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xCAFECAFE19":
      | name               | value |
      | trading.terminated | true  |
    And time is updated to "2020-01-01T01:01:01Z"

    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"

    And the orders should have the following status:
      | party | reference | status        |
      | aux   | aux-1-19  | STATUS_FILLED |
    Then the oracles broadcast data signed with "0xCAFECAFE19":
      | name             | value |
      | prices.ETH.value | 80    |

    # distribute insurance pool for DEC19
    When the network moves ahead "3" blocks
    Then the global insurance pool balance should be "0" for the asset "USD"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"

    When the oracles broadcast data signed with "0xCAFECAFE20":
      | name               | value |
      | trading.terminated | true  |
    #And time is updated to "2020-01-01T01:01:01Z"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC20"
    Then the oracles broadcast data signed with "0xCAFECAFE20":
      | name             | value |
      | prices.ETH.value | 80    |

    And the network moves ahead "3" blocks
    # When a market ETH/DEC20 is closed, the insurance pool account has its outstanding funds transferred to the [network treasury]
    And the global insurance pool balance should be "0" for the asset "USD"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | sellSideProvider | -291   | 0              | 20360        |
      | buySideProvider  | 57     | 0              | -3942        |
      | designatedLoser  | 0      | 0              | -17631       |

    Then the parties should have the following account balances:
      | party            | asset | market id | margin | general       |
      | buySideProvider  | USD   | ETH/DEC19 | 0      | 999999996044  |
      | sellSideProvider | USD   | ETH/DEC19 | 0      | 1000000020360 |
      | designatedLoser  | USD   | ETH/DEC19 | 0      | 0             |
      | aux              | USD   | ETH/DEC19 | 0      | 1000000000136 |
      | aux2             | USD   | ETH/DEC19 | 0      | 1000000000140 |
      | lpprov           | USD   | ETH/DEC19 | 0      | 1000000005301 |
