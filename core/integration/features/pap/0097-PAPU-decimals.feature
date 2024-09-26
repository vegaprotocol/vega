Feature: Protocol Automated Purchase programs - decimals

  Background:
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.fee.factors.makerFee             | 0     |
      | market.fee.factors.infrastructureFee    | 0     |
    And the average block duration is "1"


  Scenario Outline: Check automated program orders excecuting a sell order (0097-PAPU-047)(0097-PAPU-048)(0097-PAPU-049)(0097-PAPU-050)(0097-PAPU-051)(0097-PAPU-052)(0097-PAPU-053)(0097-PAPU-054)
    
    # Initialise the assets
    Given the following assets are registered:
      | id    | decimal places       |
      | base  | <baseAssetDecimals>  |
      | quote | <quoteAssetDecimals> |
    And the parties deposit on asset's general account the following amount:
      | party | asset | amount        |
      | aux1  | base  | 1000000000000 |
      | aux1  | quote | 1000000000000 |
      | aux2  | base  | 1000000000000 |
      | aux2  | quote | 1000000000000 |
    
    # Initialise the markets
    Given the spot markets:
      | id         | name       | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places  | position decimal places | sla params    |
      | base/quote | base/quote | base       | quote       | default-log-normal-risk-model | 1                | default-none | default-none     | <priceDecimals> | <positionDecimals>      | default-basic |
    And the parties place the following orders:
      | party | market id  | side | volume | price | resulting trades | type       | tif     |
      | aux1  | base/quote | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | base/quote | sell | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "base/quote"
    Then the market data for the market "base/quote" should be:
      | trading mode            | mark price |
      | TRADING_MODE_CONTINUOUS | 1000       |

    # Fund the buy back account
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount     |
      | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | base  | 1000000000 |
    When the parties submit the following one off transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type            | asset | amount     | delivery_time        |
      | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_BUY_BACK_FEES | base  | 1000000000 | 1970-01-01T00:00:00Z |
    Then the buy back fees balance should be "1000000000" for the asset "base"

    # Initialise the program
    Given the composite price oracles from "0xCAFECAFE2":
      | name         | price property | price type   | price decimals   |
      | price_oracle | prices.value   | TYPE_INTEGER | <oracleDecimals> |
    And the time triggers oracle spec is:
      | name                      | initial | every |
      | auction_schedule          | 1       | 30    |
      | auction_vol_snap_schedule | 0       | 30    |
    And the protocol automated purchase is defined as:
      | id    | from | from account type          | to account type               | market id  | price oracle | price oracle staleness tolerance | oracle offset factor | auction schedule oracle | auction volume snapshot schedule oracle | auction duration | minimum auction size | maximum auction size | expiry timestamp |
      | 12345 | base | ACCOUNT_TYPE_BUY_BACK_FEES | ACCOUNT_TYPE_NETWORK_TREASURY | base/quote | price_oracle | 60s                              | 1                    | auction_schedule        | auction_vol_snap_schedule               | 10s              | 0                    | <snapshotBalance>    | 0                |
    And the oracles broadcast data with block time signed with "0xCAFECAFE2":
      | name         | value         | time offset |
      | prices.value | <oraclePrice> | 0s          |

    # Trigger the pap snapshot
    When the network moves ahead "31" blocks
    Then the automated purchase program for market "base/quote" should have a snapshot balance of "<snapshotBalance>"

    # Trigger the pap auction
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_PROTOCOL_AUTOMATED_PURCHASE_AUCTION" for the market "base/quote"
    And the orders should have the following states:
      | party   | reference | market id  | side | status        | volume         | remaining      | price           |
      | network | 12345     | base/quote | sell | STATUS_ACTIVE | <expectedSize> | <expectedSize> | <expectedPrice> |

    # End the auction and check the network treasury balance
    Given the parties place the following orders:
      | party | market id  | side | volume         | price           | resulting trades | type       | tif     |
      | aux1  | base/quote | buy  | <expectedSize> | <expectedPrice> | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "10" blocks
    Then the network treasury balance should be "<expectedBalance>" for the asset "quote"

  Examples:
      | oracleDecimals | baseAssetDecimals | quoteAssetDecimals | priceDecimals | positionDecimals | snapshotBalance | oraclePrice | expectedSize | expectedPrice | expectedBalance |
      | 0              | 0                 | 0                  | 0             | 0                | 1000            | 100         | 1000         | 100           | 100000          |
      | 3              | 0                 | 0                  | 0             | 0                | 1000            | 100000      | 1000         | 100           | 100000          |
      | 0              | 0                 | 0                  | 3             | 0                | 1000            | 100         | 1000         | 100000        | 100000          |
      | 0              | 0                 | 0                  | 0             | 3                | 1000            | 100         | 1000000      | 100           | 100000          |
      | 0              | 0                 | 0                  | 0             | -3               | 1000            | 100         | 1            | 100           | 100000          |
      | 0              | 3                 | 0                  | 0             | 0                | 1000000         | 100         | 1000         | 100           | 100000          |
      | 0              | 0                 | 3                  | 0             | 0                | 1000            | 100         | 1000         | 100           | 100000000       |


  Scenario Outline: Check automated program orders excecuting a buy order (0097-PAPU-053)(0097-PAPU-054)(0097-PAPU-055)(0097-PAPU-056)(0097-PAPU-057)(0097-PAPU-058)(0097-PAPU-059)(0097-PAPU-060)

    # Initialise the assets
    Given the following assets are registered:
      | id    | decimal places       |
      | base  | <baseAssetDecimals>  |
      | quote | <quoteAssetDecimals> |
    And the parties deposit on asset's general account the following amount:
      | party | asset | amount        |
      | aux1  | base  | 1000000000000 |
      | aux1  | quote | 1000000000000 |
      | aux2  | base  | 1000000000000 |
      | aux2  | quote | 1000000000000 |
    
    # Initialise the markets
    Given the spot markets:
      | id         | name       | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places  | position decimal places | sla params    |
      | base/quote | base/quote | base       | quote       | default-log-normal-risk-model | 1                | default-none | default-none     | <priceDecimals> | <positionDecimals>      | default-basic |
    And the parties place the following orders:
      | party | market id  | side | volume | price | resulting trades | type       | tif     |
      | aux1  | base/quote | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | base/quote | sell | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "base/quote"
    Then the market data for the market "base/quote" should be:
      | trading mode            | mark price |
      | TRADING_MODE_CONTINUOUS | 1000       |

    # Fund the buy back account
    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount     |
      | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | quote | 1000000000 |
    When the parties submit the following one off transfers:
      | id | from                                                             | from_account_type    | to                                                               | to_account_type            | asset | amount     | delivery_time        |
      | 1  | f0b40ebdc5b92cf2cf82ff5d0c3f94085d23d5ec2d37d0b929e177c6d4d37e4c | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_BUY_BACK_FEES | quote | 1000000000 | 1970-01-01T00:00:00Z |
    Then the buy back fees balance should be "1000000000" for the asset "quote"

    # Initialise the program
    Given the composite price oracles from "0xCAFECAFE2":
      | name         | price property | price type   | price decimals   |
      | price_oracle | prices.value   | TYPE_INTEGER | <oracleDecimals> |
    And the time triggers oracle spec is:
      | name                      | initial | every |
      | auction_schedule          | 1       | 30    |
      | auction_vol_snap_schedule | 0       | 30    |
    And the protocol automated purchase is defined as:
      | id    | from  | from account type          | to account type               | market id  | price oracle | price oracle staleness tolerance | oracle offset factor | auction schedule oracle | auction volume snapshot schedule oracle | auction duration | minimum auction size | maximum auction size | expiry timestamp |
      | 12345 | quote | ACCOUNT_TYPE_BUY_BACK_FEES | ACCOUNT_TYPE_NETWORK_TREASURY | base/quote | price_oracle | 60s                              | 1                    | auction_schedule        | auction_vol_snap_schedule               | 10s              | 0                    | <snapshotBalance>    | 0                |
    And the oracles broadcast data with block time signed with "0xCAFECAFE2":
      | name         | value         | time offset |
      | prices.value | <oraclePrice> | 0s          |

    # Trigger the pap snapshot
    When the network moves ahead "31" blocks
    Then the automated purchase program for market "base/quote" should have a snapshot balance of "<snapshotBalance>"

    # Trigger the pap auction
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_PROTOCOL_AUTOMATED_PURCHASE_AUCTION" for the market "base/quote"
    And the orders should have the following states:
      | party   | reference | market id  | side | status        | volume         | remaining      | price           |
      | network | 12345     | base/quote | buy  | STATUS_ACTIVE | <expectedSize> | <expectedSize> | <expectedPrice> |

    # End the auction and check the network treasury balance
    Given the parties place the following orders:
      | party | market id  | side | volume         | price           | resulting trades | type       | tif     |
      | aux1  | base/quote | sell | <expectedSize> | <expectedPrice> | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "10" blocks
    Then the network treasury balance should be "<expectedBalance>" for the asset "base"

  Examples:
      | oracleDecimals | baseAssetDecimals | quoteAssetDecimals | priceDecimals | positionDecimals | snapshotBalance | oraclePrice | expectedSize | expectedPrice | expectedBalance |
      | 0              | 0                 | 0                  | 0             | 0                | 100000          | 100         | 1000         | 100           | 1000            |
      | 3              | 0                 | 0                  | 0             | 0                | 100000          | 100000      | 1000         | 100           | 1000            |
      | 0              | 0                 | 0                  | 3             | 0                | 100000          | 100         | 1000         | 100000        | 1000            |
      | 0              | 0                 | 0                  | 0             | 3                | 100000          | 100         | 1000000      | 100           | 1000            |
      | 0              | 0                 | 0                  | 0             | -3               | 100000          | 100         | 1            | 100           | 1000            |
      | 0              | 3                 | 0                  | 0             | 0                | 100000          | 100         | 1000         | 100           | 1000000         |
      | 0              | 0                 | 3                  | 0             | 0                | 100000000       | 100         | 1000         | 100           | 1000            |
