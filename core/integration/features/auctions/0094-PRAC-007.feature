Feature: When a market's trigger or extension_trigger is set to represent a governance suspension then no other triggers can affect the market. (0094-PRAC-007)



  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the long block duration table is:
      | threshold | duration |
      | 3s        | 1m       |
      | 40s       | 10m      |
      | 2m        | 1h       |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    # create 2 markets
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future | 2              | 0.25                   | 0                         | default-futures |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future | 2              | 0.25                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 2     |
    And the average block duration is "1"
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount        |
      | party1  | ETH   | 1000000000000 |
      | party2  | ETH   | 1000000000000 |
      | party3  | ETH   | 1000000000000 |
      | party4  | ETH   | 1000000000000 |
      | lpprov1 | ETH   | 1000000000000 |
      | lpprov2 | ETH   | 1000000000000 |
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC20 | 937000000         | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 937000000         | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party   | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov1 | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov1 | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov2 | ETH/DEC19 | 2         | 1                    | buy  | MID              | 50     | 100    |
      | lpprov2 | ETH/DEC19 | 2         | 1                    | sell | MID              | 50     | 100    |

  @LBA
  Scenario: 0094-PRAC-007: market suspended via governance is unaffected by long block auction triggers.
    # place orders and generate trades - slippage 100
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 999500  | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party1 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC20 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | party3 | ETH/DEC19 | buy  | 1      | 999500  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party3 | ETH/DEC19 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-2    |
      | party4 | ETH/DEC19 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 937000000      | 937000000    |
    And the market data for the market "ETH/DEC19" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 937000000      | 937000000    |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    #And debug detailed orderbook volumes for market "ETH/DEC20"
    And the order book should have the following volumes for market "ETH/DEC20":
      | volume | price   | side |
      | 2      | 999400  | buy  |
      | 1      | 999500  | buy  |
      | 1      | 1000000 | sell |
      | 2      | 1000100 | sell |
    #And debug detailed orderbook volumes for market "ETH/DEC19"
    And the order book should have the following volumes for market "ETH/DEC19":
      | volume | price   | side |
      | 1      | 999500  | buy  |
      | 2      | 999650  | buy  |
      | 2      | 999850  | sell |
      | 1      | 1000000 | sell |

    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party1 | 1000000 | 1    | party2 |
      | party3 | 1000000 | 1    | party4 |
    And the mark price should be "1000000" for the market "ETH/DEC20"
    And the mark price should be "1000000" for the market "ETH/DEC19"

    When the market states are updated through governance:
      | market id | state                            |
      | ETH/DEC20 | MARKET_STATE_UPDATE_TYPE_SUSPEND |

    When the previous block duration was "90s"
    Then the trading mode should be "TRADING_MODE_SUSPENDED_VIA_GOVERNANCE" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC19"

    # We know what the volume on the books look like, but let's submit some orders that will trade regardless
    # And we'll see no trades happen
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 999999 | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | party2 | ETH/DEC20 | sell | 1      | 999999 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | party3 | ETH/DEC19 | buy  | 1      | 999999 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-3    |
      | party4 | ETH/DEC19 | sell | 1      | 999999 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-2    |
    Then the trading mode should be "TRADING_MODE_SUSPENDED_VIA_GOVERNANCE" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC19"

    When the network moves ahead "1s" with block duration of "1s"
    And the market states are updated through governance:
      | market id | state                           |
      | ETH/DEC20 | MARKET_STATE_UPDATE_TYPE_RESUME |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the network moves ahead "9m50s" with block duration of "2s"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC19"

    # still in auction, 10 seconds later, though:
    When the network moves ahead "11" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party1 | 999999 | 1    | party2 |
      | party3 | 999999 | 1    | party4 |

