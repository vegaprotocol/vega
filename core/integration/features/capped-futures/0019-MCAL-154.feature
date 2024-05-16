Feature: Margin calculation on a fully collateralised capped future

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
      | network.markPriceUpdateMaximumFrequency      | 0s    |
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
      | ETH/DEC21 | ETH        | USD   | lognormal-risk-model-1 | default-capped-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 0.25                   | 0                         | default-futures | 100           | true                 | false  |

  @SLABug @NoPerp @Capped @CFC
  Scenario: 0019-MCAL-154: Party A posts an order to buy 10 contracts at a price of 30, there's no other volume in that price range so the order lands on the book and the maintenance and initial margin levels for the party and order margin account balance are all equal to 300.
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
      | aux1   | ETH/DEC21 | buy  | 2      | 9     | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |       |
      | aux2   | ETH/DEC21 | sell | 2      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |       |
      | party1 | ETH/DEC21 | buy  | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |       |
      | party2 | ETH/DEC21 | sell | 5      | 50    | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"
    And the mark price should be "50" for the market "ETH/DEC21"
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 250    | 9750    |
      | party2 | USD   | ETH/DEC21 | 250    | 9750    |
      | aux1   | USD   | ETH/DEC21 | 18     | 99982   |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 250         | 250    | 250     | 250     | cross margin |
      | party2 | ETH/DEC21 | 250         | 250    | 250     | 250     | cross margin |
      | aux1   | ETH/DEC21 | 18          | 18     | 18      | 18      | cross margin |
      | aux2   | ETH/DEC21 | 0           | 0      | 0       | 0       | cross margin |

    # The case the AC is actually about: buy 10@30
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux3  | ETH/DEC21 | buy  | 10     | 30    | 0                | TYPE_LIMIT | TIF_GTC | aux3-1    |
    And the network moves ahead "2" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 250    | 9750    |
      | party2 | USD   | ETH/DEC21 | 250    | 9750    |
      | aux1   | USD   | ETH/DEC21 | 18     | 99982   |
      | aux3   | USD   | ETH/DEC21 | 300    | 99700   |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 250         | 250    | 250     | 250     | cross margin |
      | party2 | ETH/DEC21 | 250         | 250    | 250     | 250     | cross margin |
      | aux1   | ETH/DEC21 | 18          | 18     | 18      | 18      | cross margin |
      | aux2   | ETH/DEC21 | 0           | 0      | 0       | 0       | cross margin |
      | aux3   | ETH/DEC21 | 300         | 300    | 300     | 300     | cross margin |
