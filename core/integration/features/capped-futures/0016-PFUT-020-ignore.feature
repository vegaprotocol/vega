Feature: When binary settlement is enabled, the market ignored oracle data that is neither 0 nor max price.

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the average block duration is "1"

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the settlement data decimals for the oracle named "ethDec21Oracle" is given in "0" decimal places

    And the oracle spec for settlement data filtering data from "0xCAFECAFE2" named "ethDec22Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE2" named "ethDec22Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the settlement data decimals for the oracle named "ethDec22Oracle" is given in "0" decimal places

    And the following network parameters are set:
      | name                                         | value |
      | market.auction.minimumDuration               | 1     |
      | network.markPriceUpdateMaximumFrequency      | 1s    |
      | market.liquidity.successorLaunchWindowLength | 1s    |
      | limits.markets.maxPeggedOrders               | 4     |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.02               |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600000 | 0.99        | 300               |
    And the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.0002        | 0.01 | 0  | 0.0 | 1.2   |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model             | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary |
      | ETH/DEC21 | ETH        | USD   | lognormal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle         | 0.25                   | 0                         | default-futures | 1500          | true                 | true   |
      | ETH/DEC22 | ETH        | USD   | simple-risk-model-1    | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec22Oracle         | 0.25                   | 0                         | default-futures | 1500          | false                | true   |
      | ETH/DEC23 | ETH        | USD   | lognormal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.25                   | 0                         | default-futures | 1500          | true                 | true   |

  @NoPerp @Capped @CBin
  Scenario: 0016-PFUT-020: Pass in settlement prices that are neither 0 nor max price, then settle at valid prices.
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party     | asset | amount    |
      | party1    | USD   | 10000     |
      | party2    | USD   | 10000     |
      | party3    | USD   | 10000     |
      | party4    | USD   | 10000     |
      | party5    | USD   | 10000     |
      | party6    | USD   | 10000     |
      | aux1      | USD   | 100000    |
      | aux2      | USD   | 100000    |
      | aux3      | USD   | 100000    |
      | aux4      | USD   | 100000    |
      | aux5      | USD   | 100000    |
      | aux6      | USD   | 100000    |
      | aux7      | USD   | 100000    |
      | party-lp1 | USD   | 100000000 |
      | party-lp2 | USD   | 100000000 |
      | party-lp3 | USD   | 100000000 |

    And the parties submit the following liquidity provision:
      | id  | party     | market id | commitment amount | fee | lp type    |
      | lp1 | party-lp1 | ETH/DEC21 | 30000             | 0   | submission |
      | lp2 | party-lp2 | ETH/DEC22 | 30000             | 0   | submission |
      | lp3 | party-lp3 | ETH/DEC23 | 30000             | 0   | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | aux1   | ETH/DEC21 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1-1   |       |
      | aux5   | ETH/DEC22 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-5-1   |       |
      | aux3   | ETH/DEC23 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-3-1   |       |
      | party1 | ETH/DEC21 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-p1-1  |       |
      | party3 | ETH/DEC22 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-p3-1  |       |
      | party5 | ETH/DEC23 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-p5-1  |       |
      | party2 | ETH/DEC21 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-p2-1  |       |
      | party4 | ETH/DEC22 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-p4-1  |       |
      | party6 | ETH/DEC23 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-p6-1  |       |
      | aux2   | ETH/DEC21 | sell | 2      | 1499  | 0                | TYPE_LIMIT | TIF_GTC | ref-2-1   |       |
      | aux4   | ETH/DEC23 | sell | 2      | 1499  | 0                | TYPE_LIMIT | TIF_GTC | ref-4-1   |       |
      | aux6   | ETH/DEC22 | sell | 2      | 1499  | 0                | TYPE_LIMIT | TIF_GTC | ref-6-1   |       |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC22"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC22"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC23"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC22"
    And the mark price should be "1000" for the market "ETH/DEC23"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 5000   | 5000    |
      | party2 | USD   | ETH/DEC21 | 2500   | 7500    |
      | party3 | USD   | ETH/DEC22 | 2700   | 7300    |
      | party4 | USD   | ETH/DEC22 | 2100   | 7900    |
      | party5 | USD   | ETH/DEC23 | 5000   | 5000    |
      | party6 | USD   | ETH/DEC23 | 2500   | 7500    |

    #order margin for aux1: limit price * size = 999*2=1998
    #order margin for aux2: (max price - limit price) * size = (1500-1301)*2=398
    # party1 maintenance margin level: position size * average entry price = 5*1000=5000
    # party2 maintenance margin level: position size * (max price - average entry price)=5*(1500-1000)=2500
    # Aux1: potential position * average price on book = 2 * 999 = 1998, but due to the MTM settlement the margin level
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 5000        | 5000   | 5000    | 5000    | cross margin |
      | party2 | ETH/DEC21 | 2500        | 2500   | 2500    | 2500    | cross margin |
      | aux2   | ETH/DEC21 | 2           | 2      | 2       | 2       | cross margin |
      | aux1   | ETH/DEC21 | 1998        | 1998   | 1998    | 1998    | cross margin |
      | party3 | ETH/DEC22 | 2250        | 2475   | 2700    | 3150    | cross margin |
      | party4 | ETH/DEC22 | 1750        | 1925   | 2100    | 2450    | cross margin |
      | aux5   | ETH/DEC22 | 400         | 440    | 480     | 560     | cross margin |
      | aux6   | ETH/DEC22 | 200         | 220    | 240     | 280     | cross margin |
      | party5 | ETH/DEC23 | 5000        | 5000   | 5000    | 5000    | cross margin |
      | party6 | ETH/DEC23 | 2500        | 2500   | 2500    | 2500    | cross margin |
      | aux4   | ETH/DEC23 | 2           | 2      | 2       | 2       | cross margin |
      | aux3   | ETH/DEC23 | 1998        | 1998   | 1998    | 1998    | cross margin |

    #update mark price
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | aux1-2    |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC | aux2-2    |
      | aux5  | ETH/DEC22 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | aux5-2    |
      | aux6  | ETH/DEC22 | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC | aux6-2    |
      | aux3  | ETH/DEC23 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | aux3-2    |
      | aux4  | ETH/DEC23 | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC | aux4-2    |

    And the network moves ahead "2" blocks
    Then the mark price should be "1100" for the market "ETH/DEC21"
    And the mark price should be "1100" for the market "ETH/DEC22"
    And the mark price should be "1100" for the market "ETH/DEC23"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
    And the oracles broadcast data signed with "0xCAFECAFE2":
      | name               | value |
      | trading.terminated | true  |
    Then the network moves ahead "2" blocks
    And the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    And the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC22"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 5000   | 5500    |
      | party2 | USD   | ETH/DEC21 | 2500   | 7000    |
      | party3 | USD   | ETH/DEC22 | 2970   | 7530    |
      | party4 | USD   | ETH/DEC22 | 2310   | 7190    |
      | aux1   | USD   | ETH/DEC21 | 3098   | 96908   |
      | aux2   | USD   | ETH/DEC21 | 402    | 99570   |
      | aux5   | USD   | ETH/DEC22 | 1122   | 98884   |
      | aux6   | USD   | ETH/DEC22 | 726    | 99246   |
      | party5 | USD   | ETH/DEC23 | 5000   | 5500    |
      | party6 | USD   | ETH/DEC23 | 2500   | 7000    |
      | aux3   | USD   | ETH/DEC23 | 3098   | 96908   |
      | aux4   | USD   | ETH/DEC23 | 402    | 99570   |

    # First, try settling a market via governance, providing an incorrect price.
    When the market states are updated through governance:
      | market id | state                              | settlement price | error                                       |
      | ETH/DEC23 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 123              | settlement data is outside of the price cap |
      | ETH/DEC23 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 123456789        | settlement data is outside of the price cap |
    Then the network moves ahead "1" blocks
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC23"
    # now provide a valid price
    When the market states are updated through governance:
      | market id | state                              | settlement price | error |
      | ETH/DEC23 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 0                |       |
    Then the network moves ahead "1" blocks
    And the last market state should be "STATE_CLOSED" for the market "ETH/DEC23"
    # Now we can try to settle the markets with invalid data, making sure it doesn't settle.
    # In range, but not valid for binary settlements
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 100   |
    # Out of range, so should be ignored
    When the oracles broadcast data signed with "0xCAFECAFE2":
      | name             | value |
      | prices.ETH.value | 90000 |
    And the network moves ahead "2" blocks
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    And the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC22"
    # Make sure terminated is indeed the LAST state, rather than settled
    And the last market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    And the last market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC22"

    # Now one market will settle with a valid price of 0
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 0     |
    And the network moves ahead "2" blocks
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC22"
    And the last market state should be "STATE_SETTLED" for the market "ETH/DEC21"

    # Now settle the remaining market using the max price.
    When the oracles broadcast data signed with "0xCAFECAFE2":
      | name             | value |
      | prices.ETH.value | 1500  |
    And the network moves ahead "2" blocks
    Then the last market state should be "STATE_SETTLED" for the market "ETH/DEC22"
    And the last market state should be "STATE_SETTLED" for the market "ETH/DEC21"
    # the margin balances should be empty
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 0      | 5000    |
      | party2 | USD   | ETH/DEC21 | 0      | 15000   |
      | party3 | USD   | ETH/DEC22 | 0      | 12500   |
      | party4 | USD   | ETH/DEC22 | 0      | 7500    |
      | aux1   | USD   | ETH/DEC21 | 0      | 98906   |
      | aux2   | USD   | ETH/DEC21 | 0      | 101072  |
      | aux5   | USD   | ETH/DEC22 | 0      | 100406  |
      | aux6   | USD   | ETH/DEC22 | 0      | 99572   |
      | party5 | USD   | ETH/DEC23 | 0      | 5000    |
      | party6 | USD   | ETH/DEC23 | 0      | 15000   |
      | aux3   | USD   | ETH/DEC23 | 0      | 98906   |
      | aux4   | USD   | ETH/DEC23 | 0      | 101072  |
