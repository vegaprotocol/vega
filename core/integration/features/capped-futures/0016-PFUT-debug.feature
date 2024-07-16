Feature: Settle capped futures market with a price within correct range

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
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 6              |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.02               |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600000 | 0.99        | 3                 |
    And the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau             | mu | r     | sigma |
      | 0.00001       | 0.0001140771161 | 0  | 0.016 | 0.15  |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary | decimal places | position decimal places |
      | ETH/DEC21 | ETH        | ETH   | lognormal-risk-model-fish | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 0.001                  | 0                         | default-futures | 1000          | true                 | true   | 1              | 1                       |

  @SLABug @NoPerp @Capped
  Scenario: 0016-PFUT-020: When `max_price` is specified, the `binary_settlement` flag is set to `true` and the final settlement price candidate received from the oracle is greater than `0` and less than  `max_price` the value gets ignored, next a value of `0` comes in from the settlement oracle and market settles correctly.
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount     |
      | party1   | ETH   | 1000000000 |
      | aux1     | ETH   | 1000000000 |
      | aux2     | ETH   | 1000000000 |
      | aux3     | ETH   | 1000000000 |
      | aux4     | ETH   | 1000000000 |
      | party-lp | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | lp type    |
      | lp1 | party-lp | ETH/DEC21 | 300000            | 0   | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 2      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux1  | ETH/DEC21 | buy  | 1      | 355   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux2  | ETH/DEC21 | sell | 1      | 355   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC21 | sell | 2      | 500   | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux3  | ETH/DEC21 | buy  | 150    | 397   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | aux4  | ETH/DEC21 | sell | 150    | 397   | 1                | TYPE_LIMIT | TIF_GTC | ref-6     |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 150    | 355   | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC21 | sell | 150    | 355   | 1                | TYPE_LIMIT | TIF_GTC | ref-8     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party1 | ETH/DEC21 | 967500000   |
    #margin for party1: 15*(100-35.5)=967.5

    And the parties should have the following account balances:
      | party  | asset | market id | margin    | general  |
      | party1 | ETH   | ETH/DEC21 | 967500000 | 19187500 |

    Then the following transfers should happen:
      | from   | to     | from account         | to account                       | market id | amount    | asset |
      | party1 | party1 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN              | ETH/DEC21 | 967500000 | ETH   |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 2662500   | ETH   |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 10650000  | ETH   |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 0         | ETH   |
