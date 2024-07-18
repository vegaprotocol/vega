Feature: Issue 11453: incorrect closeout due to decimals in mark price

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the average block duration is "1"
    And the following assets are registered:
      | id  | decimal places |
      | USD | 6              |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the settlement data decimals for the oracle named "ethDec21Oracle" is given in "2" decimal places

    And the following network parameters are set:
      | name                                         | value  |
      | market.auction.minimumDuration               | 1      |
      | network.markPriceUpdateMaximumFrequency      | 1s     |
      | market.liquidity.successorLaunchWindowLength | 1s     |
      | limits.markets.maxPeggedOrders               | 4      |
      | market.fee.factors.makerFee                  | 0.0002 |
      | market.fee.factors.infrastructureFee         | 0.0005 |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0002    | 0.0005             |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600000 | 0.99        | 300               |
    And the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau             | mu | r     | sigma |
      | 0.00001       | 0.0001140771161 | 0  | 0.016 | 0.15  |

    And the markets:
      | id        | quote name | asset | risk model             | margin calculator                | auction duration | fees          | price monitoring | data source config | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | decimal places | position decimal places |
      | ETH/DEC21 | ETH        | USD   | lognormal-risk-model-1 | default-capped-margin-calculator | 1                | fees-config-1 | default-none     | ethDec21Oracle     | 0.001                  | 0                         | default-futures | 1000          | true                 | true   | weight     | 1            | 1           | 5000000     | 0,1,0,0        | 1m,1m,1m,1m                | 1              | 1                       |

  @CappedBug
  Scenario: Replicate bug where MTM happens at strange price
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount        |
      | party1   | USD   | 100000000000  |
      | party2   | USD   | 100000000000  |
      | party3   | USD   | 100000000000  |
      | aux1     | USD   | 100000000000  |
      | aux2     | USD   | 100000000000  |
      | aux3     | USD   | 100000000000  |
      | aux4     | USD   | 1000000000    |
      | aux5     | USD   | 100000000000  |
      | party-lp | USD   | 1000000000000 |

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | lp type    |
      | lp2 | party-lp | ETH/DEC21 | 300000000         | 0   | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | aux1   | ETH/DEC21 | buy  | 1000   | 350   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |       |
      | aux2   | ETH/DEC21 | sell | 1000   | 450   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |       |
      | party1 | ETH/DEC21 | buy  | 1      | 400   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |       |
      | party2 | ETH/DEC21 | sell | 1      | 400   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
      | party2 | ETH/DEC21 | sell | 1000   | 500   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
      | party3 | ETH/DEC21 | buy  | 1000   | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"
    And the mark price should be "400" for the market "ETH/DEC21"
    And the parties should have the following account balances:
      | party  | asset | market id | margin     | general     |
      | party1 | USD   | ETH/DEC21 | 4000000    | 99996000000 |
      | party2 | USD   | ETH/DEC21 | 5006000000 | 94994000000 |
      | aux1   | USD   | ETH/DEC21 | 3500000000 | 96500000000 |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search     | initial    | release    | margin mode  |
      | party1 | ETH/DEC21 | 4000000     | 4000000    | 4000000    | 4000000    | cross margin |
      | party2 | ETH/DEC21 | 5006000000  | 5006000000 | 5006000000 | 5006000000 | cross margin |
      | aux1   | ETH/DEC21 | 3500000000  | 3500000000 | 3500000000 | 3500000000 | cross margin |
      | aux2   | ETH/DEC21 | 5500000000  | 5500000000 | 5500000000 | 5500000000 | cross margin |

    When the network moves ahead "2" blocks
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | aux4  | ETH/DEC21 | sell | 150    | 350   | 1                | TYPE_LIMIT | TIF_GTC | ref-4     |       |
    When the network moves ahead "2" blocks
    Then the mark price should be "400" for the market "ETH/DEC21"
    And the parties should have the following account balances:
      | party | asset | market id | margin    | general  |
      | aux4  | USD   | ETH/DEC21 | 900000000 | 24632500 |
    #And debug transfers
    #And debug detailed orderbook volumes for market "ETH/DEC21"
    #And debug trades
