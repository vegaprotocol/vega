Feature: Simple Spot Order between two parties match successfully
  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 3              |
      | BTC | 5              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.001 |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.auction.minimumDuration          | 1     |
      | validators.epoch.length                 | 60s   |
      | limits.markets.maxPeggedOrders          | 4     |
      | referralProgram.minStakedVegaTokens     | 0     |

  @SDP
  Scenario Outline: Simple Spot Order matches with counter party using different combinations of decimal places.

    Given the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | default-none     | <ETH decimals> | <BTC decimals>          | default-basic |
    # setup accounts
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000000 |
      | party2 | BTC   | 10000000 |
      | party3 | ETH   | 10000000 |
      | party4 | BTC   | 10000000 |
    # place orders and generate trades
    When the parties place the following orders:
      | party  | market id | side | volume   | price   | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | <volume> | <price> | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1111 |
      | party2 | BTC/ETH   | sell | <volume> | <price> | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1111 |

    Then "party2" should have holding account balance of "100000" for asset "BTC"
    And "party1" should have holding account balance of "10000" for asset "ETH"

    And the orders should have the following states:
      | party  | market id | side | volume   | remaining | price   | status        |
      | party1 | BTC/ETH   | buy  | <volume> | <volume>  | <price> | STATUS_ACTIVE |
      | party2 | BTC/ETH   | sell | <volume> | <volume>  | <price> | STATUS_ACTIVE |

    When the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "<mark price>" for the market "BTC/ETH"

    And the following trades should be executed:
      | buyer  | price   | size     | seller |
      | party1 | <price> | <volume> | party2 |
    And "party2" should have holding account balance of "0" for asset "BTC"
    And "party1" should have holding account balance of "0" for asset "ETH"
    When the parties place the following orders:
      | party  | market id | side | volume    | price    | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH   | buy  | <volume2> | <price2> | 0                | TYPE_LIMIT | TIF_GTC | p3-b-1    |
      | party4 | BTC/ETH   | sell | <volume2> | <price2> | 1                | TYPE_LIMIT | TIF_GTC | p4-s-1    |
    And the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer  | price    | size      | seller |
      | party3 | <price2> | <volume2> | party4 |
    And the mark price should be "<mark price 2>" for the market "BTC/ETH"
    #Then debug transfers
    And the following transfers should happen:
      | from   | from account            | to     | to account                       | market id | amount          | asset | type                                 |
      | party4 | ACCOUNT_TYPE_GENERAL    |        | ACCOUNT_TYPE_FEES_MAKER          | BTC/ETH   | <maker fee>     | ETH   | TRANSFER_TYPE_MAKER_FEE_PAY          |
      | party4 | ACCOUNT_TYPE_GENERAL    |        | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | BTC/ETH   | <infra fee>     | ETH   | TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY |
      | party4 | ACCOUNT_TYPE_GENERAL    |        | ACCOUNT_TYPE_FEES_LIQUIDITY      | BTC/ETH   | <liquidity fee> | ETH   | TRANSFER_TYPE_LIQUIDITY_FEE_PAY      |
      |        | ACCOUNT_TYPE_FEES_MAKER | party3 | ACCOUNT_TYPE_GENERAL             | BTC/ETH   | <receive fee>   | ETH   | TRANSFER_TYPE_MAKER_FEE_RECEIVE      |
    Examples:
      | ETH decimals | BTC decimals | volume | price | mark price | volume2 | price2 | mark price 2 | maker fee | infra fee | liquidity fee | receive fee |
      | 2            | 1            | 10     | 1000  | 1000       | 1       | 110    | 110          | 1         | 1         | 0             | 1           |
      | 3            | 5            | 100000 | 10000 | 10000      | 10000   | 1100   | 1100         | 1         | 1         | 0             | 1           |
      | 1            | 2            | 100    | 100   | 100        | 10      | 11     | 11           | 1         | 1         | 0             | 1           |
      | 0            | 5            | 100000 | 10    | 10         | 10000   | 1      | 1            | 1         | 1         | 0             | 1           |
      | 0            | 0            | 1      | 10    | 10         | 1       | 11     | 11           | 11        | 22        | 0             | 11          |
      | 3            | 0            | 1      | 10000 | 10000      | 1       | 11000  | 11000        | 11        | 22        | 0             | 11          |
      | 3            | 5            | 100000 | 10000 | 10000      | 100000  | 11000  | 11000        | 11        | 22        | 0             | 11          |
      | 0            | 0            | 1      | 10    | 10         | 2       | 11     | 11           | 22        | 44        | 0             | 22          |
      | 3            | 0            | 1      | 10000 | 10000      | 2       | 11000  | 11000        | 22        | 44        | 0             | 22          |
      #| 3            | 5            | 100000 | 10000 | 10000      | 10      | 10000  | 10000        | 0         | 0         | 0             | 0           |
