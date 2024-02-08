Feature: Spot market

  Scenario: 001 when an order is placed, holding account should have the holding asset; when there is not enough asset to move to holding account,
            the order can not be placed. When trade happens, holding asset is released: 

  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.value.windowLength                           | 1h    |
    
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
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 500    |
      | party3 | ETH   | 10000  |
      | party4 | BTC   | 500    |
      | party5 | ETH   | 2000   |
    And the average block duration is "2"

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GFA | party-order1111 |
      | party2 | BTC/ETH   | sell | 100    | 3000  | 0                | TYPE_LIMIT | TIF_GTC | party-order2    |
      | party1 | BTC/ETH   | buy  | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party-order11   |
      | party2 | BTC/ETH   | sell | 100    | 9000  | 0                | TYPE_LIMIT | TIF_GTC | party-order12   |
    # During opening auction, holding asset is transferred from general account to holding account:
    Then "party1" should have holding account balance of "4000" for asset "ETH"
    Then "party1" should have general account balance of "6000" for asset "ETH"
    Then "party2" should have holding account balance of "200" for asset "BTC"

    And the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | party-order2 | 1000  | 0          | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 1500       | TRADING_MODE_CONTINUOUS | 360000  | 976       | 2268      |

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

    # party2 has a sell which incurs fees, check the transfers are performed.
    Then the following transfers should happen:
      | from   | to     | from account            | to account                       | market id | amount | asset |
      | party1 | party2 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 1000   | ETH   |
      | party2 | party1 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 100    | BTC   |
      | party2 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | 10     | ETH   |
      | party2 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | 30     | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | 10     | ETH   |

    Then "party1" should have holding account balance of "1000" for asset "ETH"
    Then "party1" should have general account balance of "6510" for asset "ETH"
    Then "party1" should have general account balance of "200" for asset "BTC"

    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party2" should have general account balance of "300" for asset "BTC"
    #party2 sold 1 BTC for 10ETH, and should have 10+15=25ETH now, party2 is the price taker, so party2 subtracted 0.04*10 =0.4 from general account for the taker fee, so party2 received 24.6ETH
    Then "party2" should have general account balance of "2460" for asset "ETH"

    When the network moves ahead "2" blocks

    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 1000  | 100    |
    #If the party does not have sufficient funds in their `general` account to cover this transfer, the order should be cancelled:
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     | error                                                              |
      | party1 | BTC/ETH   | buy  | 1000   | 2000  | 0                | TYPE_LIMIT | TIF_GFN | party-order13 | party does not have sufficient balance to cover the trade and fees |
      | party5 | BTC/ETH   | buy  | 1000   | 2000  | 0                | TYPE_LIMIT | TIF_GFN | party-order13 | party does not have sufficient balance to cover the trade and fees |
      | party1 | BTC/ETH   | sell | 1000   | 2000  | 0                | TYPE_LIMIT | TIF_GFN | party-order13 | party does not have sufficient balance to cover the trade and fees |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC | party-order14 |
      | party4 | BTC/ETH   | sell | 200    | 2000  | 1                | TYPE_LIMIT | TIF_GTC | party-order15 |

    When the network moves ahead "2" blocks

    Then the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 1000  | 100    |
      | sell | 2000  | 100    |

    Then "party3" should have holding account balance of "0" for asset "ETH"
    Then "party3" should have general account balance of "8020" for asset "ETH"
    Then "party3" should have general account balance of "100" for asset "BTC"

    #the maker fee + infra fee = 0.04*(1*20)=0.8, party4 will receive 20ETH and with fee 0.8ETH subtracted, so party4 received 19.2ETH
    Then "party4" should have holding account balance of "100" for asset "BTC"
    Then "party4" should have general account balance of "1920" for asset "ETH"
    Then "party4" should have general account balance of "300" for asset "BTC"

    # If the order is cancelled or the size is reduced through an order amendment, 
    # funds should be released from the `holding_account` and returned to the `general_account`
    # (0080-SPOT-007)
    And the parties cancel the following orders:
      | party  | reference     |
      | party4 | party-order15 |
    Then "party4" should have holding account balance of "0" for asset "BTC"
    Then "party4" should have general account balance of "400" for asset "BTC"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | BTC/ETH   | buy  | 100    | 2300  | 0                | TYPE_LIMIT | TIF_GTC | party-order14 |
      | party4 | BTC/ETH   | sell | 200    | 2300  | 0                | TYPE_LIMIT | TIF_GTC | party-order15 |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                    | auction trigger       |
      | 2000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then "party3" should have holding account balance of "2335" for asset "ETH"
    Then "party4" should have holding account balance of "200" for asset "BTC"

    When the network moves ahead "2" blocks
    #past the price monitoring auction extension, so trade happened, holding account for party3 is release after trading:
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             |
      | 2300       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    Then "party3" should have holding account balance of "0" for asset "ETH"
    Then "party4" should have holding account balance of "100" for asset "BTC"


