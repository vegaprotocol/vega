Feature: Set up a spot market, with an opening auction, then uncross the book. Make sure opening auction can end.
  Background:

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 5                | fees-config-1 | default-none     |

  Scenario: set up 2 parties with balance
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 1000000000 |
      | party2 | ETH   | 1000000000 |
      | party2 | BTC   | 5          |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | party1 | BTC/ETH   | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | BTC/ETH   | sell | 5      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

    Then "party2" should have holding account balance of "5" for asset "BTC"
    Then "party1" should have holding account balance of "1000000" for asset "ETH"

    When the opening auction period ends for market "BTC/ETH"

    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party1 | 1000000 | 1    | party2 |

    And "party1" should have general account balance of "1" for asset "BTC"
    # party 2 has an order of 5 sell BTC so it would have transferred 1 to party 1 for the sell and 1 remains in the holding account
    And "party2" should have general account balance of "0" for asset "BTC"
    And "party2" should have holding account balance of "4" for asset "BTC"

    And "party1" should have general account balance of "999000000" for asset "ETH"
    # party 2 has a buy order so it also has 950000 in the holding account
    And "party2" should have holding account balance of "950000" for asset "ETH"
    And "party2" should have general account balance of "1000050000" for asset "ETH"

    And the mark price should be "1000000" for the market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # now that we're not in opening auction or any auction lets do a buy some more BTC
    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 2      | 1000000 | 1                | TYPE_LIMIT | TIF_GTC | t3-b-2    |

    # fees should be paid by the buyer
    Then "party1" should have general account balance of "3" for asset "BTC"
    # 14k paid in fees by the aggressor (party1) => 999000000 - 2000000 - 14000 =
    And "party1" should have general account balance of "996986000" for asset "ETH"

    # seller gets 2 * 1000000 ETH + 10k maker fees
    And "party2" should have holding account balance of "2" for asset "BTC"
    And "party2" should have general account balance of "1002060000" for asset "ETH"

    # now lets make the seller the aggressor, party1 now wants to sell their BTC for 950000
    # because they are paying the fees they get 950000 - fees = 950000-6650 = 943,350
    # party1 transfers 1 BTC to party2 from their general account
    # party2 gets the 950000 released from holding account and pays 943350 to party1 and 6650 fees
    # out of the 6650 ETH, 4750 ETH are maker fees which go to the general account of party2
    # therefore in total the geneal account balance of ETH for party2 after the trade is 1002060000 + 4750
    And the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | sell | 1      | 950000 | 1                | TYPE_LIMIT | TIF_GTC | t3-b-2    |

    Then "party1" should have general account balance of "2" for asset "BTC"
    # 996986000 + 943350 (950000 - fees)
    And "party1" should have general account balance of "997929350" for asset "ETH"
    And "party1" should have holding account balance of "0" for asset "ETH"

    And "party2" should have general account balance of "1" for asset "BTC"
    And "party2" should have general account balance of "1002064750" for asset "ETH"
    And "party2" should have holding account balance of "0" for asset "ETH"
    And "party2" should have holding account balance of "2" for asset "BTC"

