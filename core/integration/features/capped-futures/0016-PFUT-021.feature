Feature: When `max_price` is specified and the market is ran in a fully-collateralised mode

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
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary |
      | ETH/DEC21 | ETH        | USD   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 0.25                   | 0                         | default-futures | 1500          | true                 | false  |

  @SLABug @NoPerp @Capped
  Scenario: 0016-PFUT-021: Settlement happened when market is being closed - happens when the oracle price is < max price cap, higher prices are ignored.
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | USD   | 10000     |
      | party2   | USD   | 1000      |
      | party3   | USD   | 5000      |
      | aux1     | USD   | 100000    |
      | aux2     | USD   | 100000    |
      | party-lp | USD   | 100000000 |

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | lp type    |
      | lp2 | party-lp | ETH/DEC21 | 30000             | 0   | submission |
    And the parties place the following pegged iceberg orders:
      | party    | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party-lp | ETH/DEC21 | 600       | 30                   | buy  | BID              | 1800   | 10     |
      | party-lp | ETH/DEC21 | 600       | 30                   | sell | ASK              | 1800   | 10     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1   | ETH/DEC21 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2   | ETH/DEC21 | sell | 2      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party1 | ETH/DEC21 | buy  | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | party2 | ETH/DEC21 | sell | 5      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC21 | 1200   | 8800    |
      | party2 | USD   | ETH/DEC21 | 0      | 1000    |

    # party1 maintenance margin level: position size * average entry price
    # party2 maintenance margin level: (max price - average entry price)
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  |
      | party1 | ETH/DEC21 | 1000        | 1100   | 1200    | 1400    | cross margin |
      | party2 | ETH/DEC21 | 0           | 0      | 0       | 0       | cross margin |

