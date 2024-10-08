Feature: Spot market

  Scenario: Spot Order gets filled partially

  Background:

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 300               |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100    |
      | party2 | BTC   | 5      |

    And the average block duration is "1"
    # Then "party1" should have holding account balance of "100" for asset "ETH"
    # Then "party2" should have holding account balance of "5" for asset "BTC"

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference       | only |
      | party1 | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GFA | party-order1111 |      |
      | party2 | BTC/ETH   | sell | 1      | 30    | 0                | TYPE_LIMIT | TIF_GTC | party-order2    |      |
      | party1 | BTC/ETH   | buy  | 2      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party-order11   |      |
      | party2 | BTC/ETH   | sell | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC | party-order12   |      |

    Then "party1" should have holding account balance of "40" for asset "ETH"
    Then "party1" should have general account balance of "60" for asset "ETH"
    Then "party2" should have holding account balance of "2" for asset "BTC"

    And the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | party-order2 | 10    | 0          | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 15         | TRADING_MODE_CONTINUOUS | 360000  | 10        | 22        |


    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |

    Then "party1" should have holding account balance of "20" for asset "ETH"
    Then "party1" should have general account balance of "65" for asset "ETH"
    Then "party1" should have general account balance of "1" for asset "BTC"

    Then "party2" should have holding account balance of "1" for asset "BTC"
    #party2 sold 1BTC for 15ETH to party1
    Then "party2" should have general account balance of "3" for asset "BTC"
    Then "party2" should have general account balance of "15" for asset "ETH"

    And the parties amend the following orders:
      | party  | reference     | price | size delta | tif     |
      | party2 | party-order12 | 10    | 0          | TIF_GTC |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 10    | 1    | party2 |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    Then "party1" should have holding account balance of "10" for asset "ETH"
    Then "party1" should have general account balance of "65" for asset "ETH"
    Then "party1" should have general account balance of "2" for asset "BTC"

    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party2" should have general account balance of "3" for asset "BTC"
    #party2 sold 1 BTC for 10ETH, and should have 15+10=25ETH now
    Then "party2" should have general account balance of "25" for asset "ETH"

    When the network moves ahead "11" blocks

    #check mark price update 0008-SP-TRAD-003
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 10         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    And the parties amend the following orders:
      | party  | reference     | price | size delta | tif     | error                                                              |
      | party1 | party-order11 | 200   | 0          | TIF_GTC | party does not have sufficient balance to cover the trade and fees |

    # example for changing the target stake params
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 1.5            |
    
    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |
