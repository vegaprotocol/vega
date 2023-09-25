Feature: market orders in all market types

  # We try to exercise market orders as much as possible so that we can check
  # they work the same in both futures and perpetual markets
 
  # All order types should be able to be placed and act in the same way on a perpetual
  # market as on an expiring future market. Specifically this includes: Market orders (0014-ORDT-121)

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

  Scenario: Make sure each TIF is handled correctly
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
    # GFA market orders should not be accepted during an auction but are accepted here because the 
    # validation is performed on commands at a stage before we insert orders using the feature test framework.
   When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |        | ioc order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |        | fok order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN |        | gfn order received during auction trading |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |        |  |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |        | OrderError: Invalid Persistence |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC |        | ioc order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK |        | fok order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFN |        | gfn order received during auction trading |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |        | OrderError: Invalid Persistence |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |        |  |

   When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |        | OrderError: Invalid Expiration Datetime |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |        | OrderError: Invalid Expiration Datetime |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC23 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC23 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC23 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC23"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC23"

    # Auction only orders should be rejected
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |        | gfa order received during continuous trading |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GFA |        | gfa order received during continuous trading |


    # GTC should not be accepted for MARKET orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |        | OrderError: Invalid Persistence |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | post   | OrderError: Invalid Persistence |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC |        | OrderError: Invalid Persistence |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | post   | OrderError: Invalid Persistence |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTC | reduce | OrderError: reduce only order would not reduce position |
      
    # GTT should not be accepted for MARKET orders
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | expires in | only   | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |        | OrderError: Invalid Expiration Datetime |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | post   | OrderError: Invalid Expiration Datetime |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         |        | OrderError: Invalid Expiration Datetime |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | post   | OrderError: Invalid Expiration Datetime |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_GTT | 50         | reduce | OrderError: reduce only order would not reduce position |

    # IOC should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |         |       |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_IOC | reduce  | OrderError: reduce only order would not reduce position |

    # FOK should be accepted
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | only    | error |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_FOK |         |       |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |
      | party1| ETH/DEC23 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_FOK |         |       |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | post    | OrderError: post only order would trade |
      | party1| ETH/DEC23 | sell | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | reduce  | OrderError: reduce only order would not reduce position |


