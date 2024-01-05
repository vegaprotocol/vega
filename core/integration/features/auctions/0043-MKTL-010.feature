Feature: Ensure the markets expire if they cannot leave opening auction within the configured period


  Background:

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the following network parameters are set:
      | name                                         | value |
      | limits.markets.maxPeggedOrders               | 2     |
      | market.auction.minimumDuration               | 5     |
      | market.auction.maximumDuration               | 100s  |
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      | is passed |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 5                | my-fees-config | default-none     | default-eth-for-future | 2              | 1e6                    | 1e6                       | default-futures | true      |

  @Expires
  Scenario: Covers 0043-MKTL-010
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount        |
      | party1 | ETH   | 1000000000000 |
      | party2 | ETH   | 1000000000000 |
      | party3 | ETH   | 1000000000000 |
      | lpprov | ETH   | 1000000000000 |
    And the initial insurance pool balance is "10000" for all the markets

    # place only buy orders, we will never leave opening auction
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 937000000         | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC20 | 937000000         | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | MID              | 50     | 100    |
    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC20 | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | party1 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party3 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t3-b-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake |
      | TRADING_MODE_OPENING_AUCTION | 937000000      |
    And the insurance pool balance should be "10000" for the market "ETH/DEC20"
    And the parties should have the following account balances:
      | party  | asset | market id | margin    | general      | bond      |
      | party1 | ETH   | ETH/DEC20 | 103545373 | 999896454627 |           |
      | party2 | ETH   | ETH/DEC20 | 98368105  | 999901631895 |           |
      | party3 | ETH   | ETH/DEC20 | 103545373 | 999896454627 |           |
      | lpprov | ETH   | ETH/DEC20 | 0         | 999063000000 | 937000000 |

    # And then some more time, way past the min auction duration, but before the max duration
    When the network moves ahead "50" blocks
    # Ensure we're still in opening auction
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    And the insurance pool balance should be "10000" for the market "ETH/DEC20"
    And the market state should be "STATE_PENDING" for the market "ETH/DEC20"

    # Now move ahead past to when the market is expected to expire
    When the network moves ahead "51" blocks
    # The market should be cancelled
    Then the last market state should be "STATE_CANCELLED" for the market "ETH/DEC20"
    # The market was cancelled, the insurance pool is released instantly
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    # The insurance pool balance is accounted for, though
    And the global insurance pool balance should be "10000" for the asset "ETH"
    # Account balances have been resotred, though
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general       | bond |
      | party1 | ETH   | ETH/DEC20 | 0      | 1000000000000 |      |
      | party2 | ETH   | ETH/DEC20 | 0      | 1000000000000 |      |
      | party3 | ETH   | ETH/DEC20 | 0      | 1000000000000 |      |
      | lpprov | ETH   | ETH/DEC20 | 0      | 1000000000000 | 0    |
