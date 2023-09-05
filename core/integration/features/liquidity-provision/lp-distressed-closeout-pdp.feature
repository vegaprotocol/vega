Feature: Replicate LP getting distressed during continuous trading, and after leaving an auction

  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.bondPenaltyParameter             | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0.1   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 2     |
      | validators.epoch.length                             | 5s    |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 60          | 50            | 0.2                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 5                 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.01        | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params | position decimal places |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         | SLA        | 2                       |
    And the following network parameters are set:
      | name                                               | value |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1721       |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 1000000000 |
      | party5 | ETH   | 1000000000 |
      | party6 | ETH   | 1000000000 |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | party0 | ETH/DEC21 | 200       | 100                  | sell | MID              | 600     | 1      |
    And the parties place the following orders:
      | party  | market id | side | volume   | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 100000   | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1000     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 100000   | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1000     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

  Scenario: 001, LP gets distressed during continuous trading (0042-LIQF-014)
    When the opening auction period ends for market "ETH/DEC21"
    And the auction ends with a traded volume of "1000" at a price of "1000"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 1000           | 1000          | 900                   | 1000             | 1100                    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 720    | 1       | 1000 |
    
    # Now let's trade with LP to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 600    | 1010  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 1001  | 600  | party0 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -600   | 0              | 0            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general | bond |
      | party0 | ETH   | ETH/DEC21 | 682     | 0       |  0   |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1001       | TRADING_MODE_CONTINUOUS | 1601         | 1000           | 1600          | 900                   | 1000             | 1100                    |
    
    # Raise mark price so that the LP gets liquidated
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/DEC21 | buy  | 100    | 1055  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 100    | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | open interest |
      | 1055       | TRADING_MODE_CONTINUOUS | 1700          |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/DEC21 | 0           |
    
    When the network moves ahead "7" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 1000              | STATUS_CANCELLED |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | 1055       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 1793         | 0              | 1700          |
    And the insurance pool balance should be "1152" for the market "ETH/DEC21"

  Scenario: 002, LP gets distressed after auction
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp2 | party6 | ETH/DEC21 | 1000              | 0.001 | submission |
      | lp2 | party6 | ETH/DEC21 | 1000              | 0.001 |            |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | party6 | ETH/DEC21 | 200       | 100                  | buy  | MID              | 60000   | 100    |
      | party6 | ETH/DEC21 | 200       | 100                  | sell | MID              | 60000   | 100    |
    And the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1000" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 2000           | 1000          | 900                   | 1000             | 1100                    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 720    | 1       | 1000 |
   
    # Now let's trade with LP1 to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 600    | 1010  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 1001  | 600  | party0 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -600   | 0              | 0            |
    # LP1 margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general | bond |
      | party0 | ETH   | ETH/DEC21 | 682     | 0       |  0   |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price | horizon | min bound | max bound |
      | 1001       | TRADING_MODE_CONTINUOUS | 1601         | 2000           | 1600          | 900                   | 1000             | 1100                    | 1       | 950       | 1060      |
   
    # Generate a trade outwith price monitoring bounds so that LP1 gets liquidated upon auction uncrossing
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/DEC21 | buy  | 100    | 1061  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 100    | 1061  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | open interest |
      | 1001       | TRADING_MODE_MONITORING_AUCTION | 1600          |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 682      | 0       | 0    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/DEC21 | 1195        |
    
    When the network moves ahead "7" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode             | auction trigger             | target stake | supplied stake | open interest |
      | 1061       | TRADING_MODE_CONTINUOUS  | AUCTION_TRIGGER_UNSPECIFIED | 1803         | 1000           | 1700          |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 3       | 0    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/DEC21 | 0           |
    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 1000              | STATUS_CANCELLED |
    And the insurance pool balance should be "1152" for the market "ETH/DEC21"

  Scenario: 003, 2 LPs on the market, LP1 gets distressed and closed-out during continuous trading (0042-LIQF-014)
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp2 | party6 | ETH/DEC21 | 1000              | 0.001 | submission |
      | lp2 | party6 | ETH/DEC21 | 1000              | 0.001 |            |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | party6 | ETH/DEC21 | 200       | 100                  | buy  | MID              | 60000   | 100    |
      | party6 | ETH/DEC21 | 200       | 100                  | sell | MID              | 60000   | 100    |
    And the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1000" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 2000           | 1000          | 900                   | 1000             | 1100                    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 720    | 1       | 1000 |
   
    # Now let's trade with LP1 to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 600    | 1010  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 1001  | 600  | party0 |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -600   | 0              | 0            |
    # LP1 margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general | bond |
      | party0 | ETH   | ETH/DEC21 | 682     | 0       |  0   |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1001       | TRADING_MODE_CONTINUOUS | 1601         | 2000           | 1600          | 900                   | 1000             | 1100                    |
    
    # Raise mark price so that the LP1 gets liquidated
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/DEC21 | buy  | 100    | 1055  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 100    | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | open interest |
      | 1055       | TRADING_MODE_CONTINUOUS | 1700          |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/DEC21 | 0           |
    
    When the network moves ahead "7" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 1000              | STATUS_CANCELLED |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode             | auction trigger             | target stake | supplied stake | open interest |
      | 1055       | TRADING_MODE_CONTINUOUS  | AUCTION_TRIGGER_UNSPECIFIED | 1793         | 1000           | 1700          |
    And the insurance pool balance should be "1152" for the market "ETH/DEC21"
