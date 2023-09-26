Feature: iceberg orders in all market types

  # We try to exercise iceberg orders as much as possible so that we can check
  # they work the same in both futures and perpetual markets
 
  # All order types should be able to be placed and act in the same way on a perpetual
  # market as on an expiring future market. Specifically this includes: Iceberg orders (0014-ORDT-122)

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC23 | ETH        | ETH   | default-simple-risk-model-3   | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | spam.protection.max.stopOrdersPerMarket | 5     |

  Scenario:1 Make sure each TIF is handled correctly all in one block
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |
      | party4 | ETH   | 10000000 |
      | lp1    | ETH   | 10000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lp1    | ETH/DEC23 | 10000000          | 0.1 | submission |

    # We are in auction now so make sure orders act correctly
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 10     | 899   | 0                | TYPE_LIMIT | TIF_GTC | 9         | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 10     | 899   | 0                | TYPE_LIMIT | TIF_IOC | 9         | 1                    |        | ioc order received during auction trading |      
      | party1 | ETH/DEC23 | buy  | 10     | 899   | 0                | TYPE_LIMIT | TIF_FOK | 9         | 1                    |        | fok order received during auction trading |      
      | party1 | ETH/DEC23 | buy  | 10     | 899   | 0                | TYPE_LIMIT | TIF_GFA | 9         | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 10     | 899   | 0                | TYPE_LIMIT | TIF_GFN | 9         | 1                    |        | gfn order received during auction trading |      
      | party1 | ETH/DEC23 | sell | 10     | 1101  | 0                | TYPE_LIMIT | TIF_GTC | 9         | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 10     | 1101  | 0                | TYPE_LIMIT | TIF_IOC | 9         | 1                    |        | ioc order received during auction trading |      
      | party1 | ETH/DEC23 | sell | 10     | 1101  | 0                | TYPE_LIMIT | TIF_FOK | 9         | 1                    |        | fok order received during auction trading |      
      | party1 | ETH/DEC23 | sell | 10     | 1101  | 0                | TYPE_LIMIT | TIF_GFA | 9         | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 10     | 1101  | 0                | TYPE_LIMIT | TIF_GFN | 9         | 1                    |        | gfn order received during auction trading |      

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 10     | 899   | 0                | TYPE_LIMIT | TIF_GTT | 50         | 9         | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 10     | 1101  | 0                | TYPE_LIMIT | TIF_GTT | 50         | 9         | 1                    |        |       |      

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "10" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # We reject orders of type MARKET 
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | 9         | 1                    |        | OrderError: Invalid Persistence |      
      | party1 | ETH/DEC23 | sell | 10     | 0     | 0                | TYPE_MARKET | TIF_GTC | 9         | 1                    |        | OrderError: Invalid Persistence |      
      | party1 | ETH/DEC23 | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | 9         | 1                    |        |  |      
      | party1 | ETH/DEC23 | sell | 10     | 0     | 1                | TYPE_MARKET | TIF_IOC | 9         | 1                    |        |  |      
      | party1 | ETH/DEC23 | buy  | 10     | 0     | 1                | TYPE_MARKET | TIF_FOK | 9         | 1                    |        |  |      
      | party1 | ETH/DEC23 | sell | 10     | 0     | 1                | TYPE_MARKET | TIF_FOK | 9         | 1                    |        |  |      

    # We can have IOC icebergs 
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_IOC | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_IOC | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_IOC | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_IOC | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_IOC | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_IOC | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      

    # We can have FOK icebergs 
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_FOK | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_FOK | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_FOK | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_FOK | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_FOK | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_FOK | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      

    # We can have GTC icebergs
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      

    # We can have GTT icebergs
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTT | 50         | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTT | 50         | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      

    # We can have GFN icebergs
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GFN | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GFN | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GFN | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GFN | 90        | 1                    |        |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GFN | 90        | 1                    | post   |       |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GFN | 90        | 1                    | reduce | OrderError: reduce only order would not reduce position |      

    # We can't have GFA icebergs in continuous trading
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only   | error |
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GFA | 90        | 1                    |        | gfa order received during continuous trading |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GFA | 90        | 1                    | post   | gfa order received during continuous trading |      
      | party1 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GFA | 90        | 1                    | reduce | gfa order received during continuous trading |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GFA | 90        | 1                    |        | gfa order received during continuous trading |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GFA | 90        | 1                    | post   | gfa order received during continuous trading |      
      | party1 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GFA | 90        | 1                    | reduce | gfa order received during continuous trading |      


