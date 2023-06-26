Feature: Spot market

  Scenario: Spot Order gets filled partially

  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring | decimal places | position decimal places |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | default-none     | 2              | 2                       |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 500    |
    And the average block duration is "2"

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GFA | party-order1111 |
      | party2 | BTC/ETH   | sell | 100    | 3000  | 0                | TYPE_LIMIT | TIF_GTC | party-order2    |
      | party1 | BTC/ETH   | buy  | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party-order11   |
      | party2 | BTC/ETH   | sell | 100    | 9000  | 0                | TYPE_LIMIT | TIF_GTC | party-order12   |

    Then "party1" should have holding account balance of "4000" for asset "ETH"
    Then "party1" should have general account balance of "6000" for asset "ETH"
    Then "party2" should have holding account balance of "200" for asset "BTC"

    And the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | party-order2 | 1000  | 0          | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 1500       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1500  | 100  | party2 |
    Then the network moves ahead "10" blocks

    Then "party1" should have holding account balance of "2000" for asset "ETH"
    Then "party1" should have general account balance of "6500" for asset "ETH"
    Then "party1" should have general account balance of "100" for asset "BTC"

    Then "party2" should have holding account balance of "100" for asset "BTC"
    #party2 sold 1BTC for 15ETH to party1
    Then "party2" should have general account balance of "300" for asset "BTC"
    Then "party2" should have general account balance of "1500" for asset "ETH"


    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 1000  | 200    |
      | sell | 9000  | 100    |

    And the parties amend the following orders:
      | party  | reference     | price | size delta | tif     |
      | party2 | party-order12 | 1000  | 0          | TIF_GTC |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 100  | party2 |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 1500       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    Then debug transfers
    Then the following transfers should happen:
      | from   | to     | from account            | to account                       | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 10     | ETH   |
      | party1 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 30     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL | BTC/ETH | 10 | ETH |

    Then "party1" should have holding account balance of "1000" for asset "ETH"
    Then "party1" should have general account balance of "6510" for asset "ETH"
    Then "party1" should have general account balance of "200" for asset "BTC"

    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party2" should have general account balance of "300" for asset "BTC"
    #party2 sold 1 BTC for 10ETH, and should have 15+10=25ETH now
    Then "party2" should have general account balance of "2460" for asset "ETH"

    When the network moves ahead "2" blocks