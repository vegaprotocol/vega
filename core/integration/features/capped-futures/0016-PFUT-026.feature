Feature: Futures market can be created with a with [hardcoded risk factors](./0018-RSKM-quant_risk_models.ipynb).

  Background:
    Given the average block duration is "1"

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the settlement data decimals for the oracle named "ethDec21Oracle" is given in "0" decimal places

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
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.2   | 100         | -100          | 0.2                    |
    And the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.0002        | 0.01 | 0  | 0.0 | 1.2   |
    And the markets:
      | id        | quote name | asset | risk model        | margin calculator                | auction duration | fees          | price monitoring | data source config | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary |
      | ETH/DEC21 | ETH        | USD   | simple-risk-model | default-capped-margin-calculator | 1                | fees-config-1 | default-none     | ethDec21Oracle     | 0.25                   | 0                         | default-futures | 1500          | false                | false  |

  @SLABug @NoPerp @Capped @MarginCap
  Scenario: 0016-PFUT-026, 0016-PFUT-028: Capped futures market can be created with a with [hardcoded risk factors](./0018-RSKM-quant_risk_models.ipynb).
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | USD   | 10000     |
      | party2   | USD   | 10000     |
      | aux1     | USD   | 100000    |
      | aux2     | USD   | 100000    |
      | aux3     | USD   | 100000    |
      | aux4     | USD   | 100000    |
      | aux5     | USD   | 100000    |
      | party-lp | USD   | 100000000 |

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | lp type    |
      | lp2 | party-lp | ETH/DEC21 | 30000             | 0   | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | aux1   | ETH/DEC21 | buy  | 2      | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |       |
      | aux2   | ETH/DEC21 | sell | 2      | 1499  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |       |
      | party1 | ETH/DEC21 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |       |
      | party2 | ETH/DEC21 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 1750   | 8250    |
      | party2 | USD   | ETH/DEC21 | 2250   | 7750    |

    #order margin for aux1: limit price * size = 999*2=1998
    #order margin for aux2: (max price - limit price) * size = (1500-1301)*2=398
    #party1 maintenance margin level: position size * average entry price = 5*1000=5000
    #party2 maintenance margin level: position size * (max price - average entry price)=5*(1500-1000)=2500
    #aux1: potential position * average price on book = 2 * 0 = 0
    #aux2: potential position * (max price -  limit price) = 2 * 0 = 0
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 1750        | 1750   | 1750    | 1750    | cross margin |
      | party2 | ETH/DEC21 | 2250        | 2250   | 2250    | 2250    | cross margin |
      | aux2   | ETH/DEC21 | 400         | 400    | 400     | 400     | cross margin |
      | aux1   | ETH/DEC21 | 200         | 200    | 200     | 200     | cross margin |

    #update mark price
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux1-2    |
      | aux2  | ETH/DEC21 | sell | 1      | 1     | 1                | TYPE_LIMIT | TIF_GTC | aux2-2    |

    And the network moves ahead "2" blocks
    Then the mark price should be "1" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer | price | size | seller |
      | aux1  | 1     | 1    | aux2   |

    # MTM settlement 5 long makes a loss of 5000, 5 short makes a profit of 5000
    # Now for aux1 and 2, the calculations from above still hold but more margin is required due to the open positions:
    # aux1: position * 1100 + 999*2 = 1100 + 1998 = 3098
    # aux2: then placing the order (max price - average order price) * 3 = (1500 - (1301 + 1301 + 1100)/3) * 3 = (1500 - 1234) * 3 = 266 * 3 = 798
    # aux2's short position and potential margins are calculated separately as 2 * (1500-1301) + 1 * (1500 - 1100) = 398 + 400 = 798
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 2      | 5003    |
      | party2 | USD   | ETH/DEC21 | 3      | 14992   |
      | aux1   | USD   | ETH/DEC21 | 1      | 100000  |
      | aux2   | USD   | ETH/DEC21 | 1      | 99997   |

    #update mark price to max_price
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | aux3  | ETH/DEC21 | buy  | 2      | 1499  | 1                | TYPE_LIMIT | TIF_GTC | aux3-1    |       |

    And the following trades should be executed:
      | buyer | price | size | seller |
      | aux3  | 1499  | 2    | aux2   |

    And the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then the mark price should be "1499" for the market "ETH/DEC21"

    # MTM settlement 5 long makes a profit of 2000, 5 short loses 2000
    # Now for aux1 and 2, the calculations from above still hold but more margin is required due to the open positions:
    # aux1: position * 1100 + 999*2 = 1100 + 1998 = 3098
    # aux2: short position of size 2, traded price at 1500, then margin: postion size * (max price - average entry price) = 3*(1100+1500*2)/3
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 2624   | 9871    |
      | party2 | USD   | ETH/DEC21 | 3373   | 4132    |
      | aux1   | USD   | ETH/DEC21 | 825    | 100674  |
      | aux2   | USD   | ETH/DEC21 | 2024   | 96491   |
      | aux3   | USD   | ETH/DEC21 | 1050   | 98875   |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 2624        | 2624   | 2624    | 2624    | cross margin |
      | party2 | ETH/DEC21 | 3373        | 3373   | 3373    | 3373    | cross margin |
      | aux2   | ETH/DEC21 | 2024        | 2024   | 2024    | 2024    | cross margin |
      | aux1   | ETH/DEC21 | 825         | 825    | 825     | 825     | cross margin |

    When the markets are updated:
      | id        | risk model             |
      | ETH/DEC21 | lognormal-risk-model-1 |
    And the network moves ahead "2" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 2624        | 2624   | 2624    | 2624    | cross margin |
      | party2 | ETH/DEC21 | 3373        | 3373   | 3373    | 3373    | cross margin |
      | aux2   | ETH/DEC21 | 2024        | 2024   | 2024    | 2024    | cross margin |
      | aux1   | ETH/DEC21 | 825         | 825    | 825     | 825     | cross margin |
