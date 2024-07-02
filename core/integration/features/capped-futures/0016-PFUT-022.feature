Feature: When max_price is specified and the market is ran in a fully-collateralised mode and it has parties with open positions settling it at a price of 0 works correctly

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

    And the markets:
      | id        | quote name | asset | risk model             | margin calculator                | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary |
      | ETH/DEC21 | ETH        | USD   | lognormal-risk-model-1 | default-capped-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 0.25                   | 0                         | default-futures | 1500          | true                 | false  |

  @SLABug @NoPerp @Capped @CSettle
  Scenario: 0016-PFUT-022: The sum of all final settlement cashflows equals 0 (loss socialisation does not happen). Assuming general account balances of all parties were 0 after opening the positions and all of their funds were in the margin accounts: short parties end up with balances equal to abs(position size) * max_price and long parties end up with 0 balances.
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
      | aux1   | ETH/DEC21 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |       |
      | aux2   | ETH/DEC21 | sell | 2      | 1499  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |       |
      | party1 | ETH/DEC21 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |       |
      | party2 | ETH/DEC21 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 5000   | 5000    |
      | party2 | USD   | ETH/DEC21 | 2500   | 7500    |

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

    #update mark price
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | aux1-2    |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 1                | TYPE_LIMIT | TIF_GTC | aux2-2    |

    And the network moves ahead "2" blocks
    Then the mark price should be "1100" for the market "ETH/DEC21"

    # MTM settlement 5 long makes a profit of 500, 5 short loses 500
    # Now for aux1 and 2, the calculations from above still hold but more margin is required due to the open positions:
    # aux1: position * 1100 + 999*2 = 1100 + 1998 = 3098
    # aux2: then placing the order (max price - average order price) * 3 = (1500 - (1301 + 1301 + 1100)/3) * 3 = (1500 - 1234) * 3 = 266 * 3 = 798
    # aux2's short position and potential margins are calculated separately as 2 * (1500-1301) + 1 * (1500 - 1100) = 398 + 400 = 798
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 5000   | 5500    |
      | party2 | USD   | ETH/DEC21 | 2500   | 7000    |
      | aux1   | USD   | ETH/DEC21 | 3098   | 96908   |
      | aux2   | USD   | ETH/DEC21 | 402    | 99570   |
    # The market is fully collateralised, switching to isolated margin is not supported
    When the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                                                                                                 |
      | party1 | ETH/DEC21 | isolated margin | 0.5           | margin factor (0.5) must be greater than max(riskFactorLong (0.3696680542085883), riskFactorShort (0.5650462045113667)) + linearSlippageFactor (0.25) |
      | party1 | ETH/DEC21 | isolated margin | 0.9           | isolated margin not permitted on fully collateralised markets                                                                                         |

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
      | party1 | USD   | ETH/DEC21 | 5000   | 7495    |
      | party2 | USD   | ETH/DEC21 | 2500   | 5005    |
      | aux1   | USD   | ETH/DEC21 | 3098   | 97307   |
      | aux2   | USD   | ETH/DEC21 | 402    | 99186   |
      | aux3   | USD   | ETH/DEC21 | 2998   | 96927   |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 5000        | 5000   | 5000    | 5000    | cross margin |
      | party2 | ETH/DEC21 | 2500        | 2500   | 2500    | 2500    | cross margin |
      | aux2   | ETH/DEC21 | 402         | 402    | 402     | 402     | cross margin |
      | aux1   | ETH/DEC21 | 3098        | 3098   | 3098    | 3098    | cross margin |
    #trade at max_price
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux4  | ETH/DEC21 | buy  | 2      | 1499  | 0                | TYPE_LIMIT | TIF_GTC | aux4-1    |
      | aux5  | ETH/DEC21 | sell | 2      | 1499  | 1                | TYPE_LIMIT | TIF_GTC | aux5-1    |

    And the network moves ahead "2" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux4  | USD   | ETH/DEC21 | 2998   | 97017   |
      | aux5  | USD   | ETH/DEC21 | 2      | 99923   |
    And the following transfers should happen:
      | from   | to   | from account            | to account                       | market id | amount | asset |
      | aux5   |      | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 15     | USD   |
      | aux5   |      | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 60     | USD   |
      | aux5   |      | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 0      | USD   |
      | market | aux4 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 15     | USD   |

    # Terminate trading
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
    And the network moves ahead "2" blocks
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    # Trigger settlement at price 0.
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 0     |
    And the network moves ahead "2" blocks
    Then the last market state should be "STATE_SETTLED" for the market "ETH/DEC21"
    And the following transfers should happen:
      | from     | to       | from account            | to account              | market id | amount | asset | type                        |
      | aux1     |          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC21 | 1499   | USD   | TRANSFER_TYPE_LOSS          |
      | aux3     |          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC21 | 2998   | USD   | TRANSFER_TYPE_LOSS          |
      | aux4     |          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC21 | 2998   | USD   | TRANSFER_TYPE_LOSS          |
      | party1   |          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC21 | 5000   | USD   | TRANSFER_TYPE_LOSS          |
      | party1   |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC21 | 2495   | USD   | TRANSFER_TYPE_LOSS          |
      | market   | aux2     | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC21 | 4497   | USD   | TRANSFER_TYPE_WIN           |
      | market   | aux5     | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC21 | 2998   | USD   | TRANSFER_TYPE_WIN           |
      | market   | party2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC21 | 7495   | USD   | TRANSFER_TYPE_WIN           |
      | aux1     | aux1     | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL    | ETH/DEC21 | 1599   | USD   | TRANSFER_TYPE_CLEAR_ACCOUNT |
      | aux2     | aux2     | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL    | ETH/DEC21 | 4899   | USD   | TRANSFER_TYPE_CLEAR_ACCOUNT |
      | aux5     | aux5     | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL    | ETH/DEC21 | 3000   | USD   | TRANSFER_TYPE_CLEAR_ACCOUNT |
      | party2   | party2   | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL    | ETH/DEC21 | 9995   | USD   | TRANSFER_TYPE_CLEAR_ACCOUNT |
      | party-lp | party-lp | ACCOUNT_TYPE_BOND       | ACCOUNT_TYPE_GENERAL    | ETH/DEC21 | 30000  | USD   | TRANSFER_TYPE_CLEAR_ACCOUNT |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 0      | 5000    |
      | party2 | USD   | ETH/DEC21 | 0      | 15000   |
      | aux1   | USD   | ETH/DEC21 | 0      | 98906   |
      | aux2   | USD   | ETH/DEC21 | 0      | 104085  |
      | aux3   | USD   | ETH/DEC21 | 0      | 96927   |
      | aux4   | USD   | ETH/DEC21 | 0      | 97017   |
      | aux5   | USD   | ETH/DEC21 | 0      | 102923  |
