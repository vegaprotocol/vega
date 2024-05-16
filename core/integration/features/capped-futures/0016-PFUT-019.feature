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
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 0.25                   | 0                         | default-futures | 1500          | false                | false  |


  @SLABug @NoPerp @Capped
  Scenario: 0016-PFUT-019: When `max_price` is specified and the final settlement price candidate received from the oracle is equal to `max_price` comes and market settles correctly
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | party1   | ETH   | 10000     |
      | party2   | ETH   | 1000      |
      | party3   | ETH   | 5000      |
      | aux1     | ETH   | 100000    |
      | aux2     | ETH   | 100000    |
      | party-lp | ETH   | 100000000 |

    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | lp type    |
      | lp2 | party-lp | ETH/DEC21 | 3000000           | 0   | submission |
    And the parties place the following pegged iceberg orders:
      | party    | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party-lp | ETH/DEC21 | 60000     | 30000                | buy  | BID              | 180000 | 10     |
      | party-lp | ETH/DEC21 | 60000     | 30000                | sell | ASK              | 180000 | 10     |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | buy  | 2      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |
      | aux2  | ETH/DEC21 | sell | 2      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-8     |
    And the network moves ahead "2" blocks

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the market state should be "STATE_ACTIVE" for the market "ETH/DEC21"

    Then the network moves ahead "2" blocks

    # The market considered here ("ETH/DEC19") relies on "0xCAFECAFE" oracle, checking that broadcasting events from "0xCAFECAFE1" should have no effect on it apart from insurance pool transfer
    And the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And the network moves ahead "2" blocks

    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 2000  |

    And the network moves ahead "2" blocks
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"
    # Make sure this does not settle the market
    And the last market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    # within the price range
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 1500  |

    And the network moves ahead "2" blocks
    # Now the market shows as settled
    Then the last market state should be "STATE_SETTLED" for the market "ETH/DEC21"


