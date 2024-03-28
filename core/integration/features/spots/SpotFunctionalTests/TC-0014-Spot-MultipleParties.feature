Feature: Spot order gets filled partially by 10 different parties

  Scenario: Spot Order gets filled partially by 10 different parties

  Background:

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |

    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36000   | 0.999       | 3                 |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | party1  | ETH   | 100    |
      | party2  | BTC   | 5      |
      | party3  | ETH   | 200    |
      | party4  | ETH   | 50     |
      | party5  | ETH   | 10     |
      | party6  | ETH   | 30     |
      | party7  | BTC   | 7      |
      | party8  | ETH   | 100    |
      | party9  | BTC   | 3      |
      | party10 | BTC   | 9      |

    # place orders and generate trades
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party4  | BTC/ETH   | buy  | 1      | 3     | 0                | TYPE_LIMIT | TIF_GTC | party-order4  |
      | party5  | BTC/ETH   | buy  | 1      | 5     | 0                | TYPE_LIMIT | TIF_GTC | party-order5  |
      | party6  | BTC/ETH   | buy  | 1      | 12    | 0                | TYPE_LIMIT | TIF_GTC | party-order6  |
      | party8  | BTC/ETH   | buy  | 5      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party-order8  |
      | party3  | BTC/ETH   | buy  | 5      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party-order3  |
      | party1  | BTC/ETH   | buy  | 2      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party-order11 |
      | party1  | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GFA | party-order1  |
      | party2  | BTC/ETH   | sell | 1      | 30    | 0                | TYPE_LIMIT | TIF_GTC | party-order2  |
      | party7  | BTC/ETH   | sell | 2      | 60    | 0                | TYPE_LIMIT | TIF_GTC | party-order7  |
      | party2  | BTC/ETH   | sell | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC | party-order12 |
      | party9  | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | party-order9  |
      | party10 | BTC/ETH   | sell | 5      | 200   | 0                | TYPE_LIMIT | TIF_GTC | party-order10 |


    Then "party1" should have holding account balance of "40" for asset "ETH"
    Then "party2" should have holding account balance of "2" for asset "BTC"
    Then "party3" should have holding account balance of "50" for asset "ETH"
    Then "party4" should have holding account balance of "3" for asset "ETH"
    Then "party5" should have holding account balance of "5" for asset "ETH"
    Then "party6" should have holding account balance of "12" for asset "ETH"
    Then "party7" should have holding account balance of "2" for asset "BTC"
    Then "party8" should have holding account balance of "50" for asset "ETH"
    Then "party9" should have holding account balance of "1" for asset "BTC"
    Then "party10" should have holding account balance of "5" for asset "BTC"

    And the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | party-order2 | 10    | 0          | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 36000   | 14        | 17        | 0            | 0              | 0             |

    Then "party1" should have holding account balance of "20" for asset "ETH"
    Then "party2" should have holding account balance of "1" for asset "BTC"
    Then "party7" should have holding account balance of "2" for asset "BTC"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |

    And the parties amend the following orders:
      | party  | reference     | price | size delta | tif     |
      | party2 | party-order12 | 10    | 0          | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 15         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then "party1" should have holding account balance of "21" for asset "ETH"
    Then "party2" should have holding account balance of "1" for asset "BTC"
    Then "party3" should have holding account balance of "51" for asset "ETH"
    Then "party4" should have holding account balance of "4" for asset "ETH"
    Then "party5" should have holding account balance of "6" for asset "ETH"
    Then "party6" should have holding account balance of "13" for asset "ETH"
    Then "party7" should have holding account balance of "2" for asset "BTC"
    Then "party8" should have holding account balance of "51" for asset "ETH"
    Then "party9" should have holding account balance of "1" for asset "BTC"
    Then "party10" should have holding account balance of "5" for asset "BTC"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 15         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    And the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party7 | party-order7 | 10    | 0          | TIF_GTC |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 15         | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then "party7" should have holding account balance of "3" for asset "BTC"

# as we're amending sell orders the price change doesn't affect their ability to cover the trade, only the size matters,
And the parties amend the following orders:
  | party   | reference     | price | size delta | tif     | error                                                        |
  | party10 | party-order10 | 10    | 5          | TIF_GTC | party does not have sufficient balance to cover the new size |
  | party9  | party-order9  | 10    | 3          | TIF_GTC | party does not have sufficient balance to cover the new size |

Then "party1" should have holding account balance of "21" for asset "ETH"
Then "party2" should have holding account balance of "1" for asset "BTC"
Then "party3" should have holding account balance of "51" for asset "ETH"
Then "party4" should have holding account balance of "4" for asset "ETH"
Then "party5" should have holding account balance of "6" for asset "ETH"
Then "party6" should have holding account balance of "13" for asset "ETH"
Then "party7" should have holding account balance of "3" for asset "BTC"
Then "party8" should have holding account balance of "51" for asset "ETH"
Then "party9" should have holding account balance of "1" for asset "BTC"
Then "party10" should have holding account balance of "5" for asset "BTC"