Feature: Fees reward calculations for a single asset, single market

  Background:
    Given the following network parameters are set:
      | name                                              | value  |
      | reward.asset                                      | VEGA   |
      | validators.epoch.length                           | 10s    |
      | validators.delegation.minAmount                   | 10     |
      | reward.staking.delegation.delegatorShare          | 0.883  |
      | reward.staking.delegation.minimumValidatorStake   | 100    |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000 |
      | reward.staking.delegation.competitionLevel        | 1.1    |
      | reward.staking.delegation.minValidators           | 5      |
      | reward.staking.delegation.optimalStakeMultiplier  | 5.0    |
      | network.markPriceUpdateMaximumFrequency           | 0s    |

    Given time is updated to "2021-08-26T00:00:00Z"
    Given the average block duration is "2"

     #complete the epoch to advance to a meaningful epoch (can't setup transfer to start at epoch 0)
    Then the network moves ahead "7" blocks

  Scenario: Testing fees in continuous trading with one trade and no liquidity providers - testing maker fee received
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset  | amount   |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | VEGA   | 1000000  |

    # setup recurring transfer to the maker fee reward account - this will start at the end of this epoch (1)
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                                | to_account_type                         | asset  | amount | start_epoch | end_epoch | factor | metric                               | metric_asset | markets |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000  | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA   | 10000  |       1     |           |    1   |  DISPATCH_METRIC_MAKER_FEES_RECEIVED |      ETH     |         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000     |
      | trader4 | ETH   | 10000     |
      | lpprov  | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 720    | 9280    |

    And the accumulated infrastructure fees should be "0" for the asset "ETH"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer   | price | size | seller  | aggressor side |
      | trader3 | 1002  | 3    | trader4 | sell           |

    # trade_value_for_fee_purposes = size_of_trade 0000000000000000000000000000000000000000000000000000000000000000 price_of_trade = 3 00000000000000000000000000000000000000000000000000000000000000001002 = 3006
    # infrastructure_fee = fee_factor[infrastructure] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.002 0000000000000000000000000000000000000000000000000000000000000000 3006 = 6.012 = 7 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.005 0000000000000000000000000000000000000000000000000000000000000000 3006 = 15.030 = 16 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0 0000000000000000000000000000000000000000000000000000000000000000 3006 = 0

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 16     | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 301    | ETH   |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 16     | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 7 + 16 + 0 = 23
    # Trader3 margin + general account balance = 10000 + 16 ( Maker fees) = 10016
    # Trader4 margin + general account balance = 10000 - 16 ( Maker fees) - 7 (Infra fee) = 99977

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 1089   | 8927    |
      | trader4 | ETH   | ETH/DEC21 | 715    | 8961    |

    And the accumulated infrastructure fees should be "7" for the asset "ETH"
    And the accumulated liquidity fees should be "301" for the market "ETH/DEC21"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks
    # only trader3 received the maker fees so only they get the reward of 10k
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 1089   | 8927    |
      | trader4 | ETH   | ETH/DEC21 | 715    | 8961    |

    Then "trader3" should have general account balance of "10000" for asset "VEGA"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks
    # expect no change to anyone
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 1089   | 8927    |
      | trader4 | ETH   | ETH/DEC21 | 715    | 8961    |

    Then "trader3" should have general account balance of "10000" for asset "VEGA"

  Scenario: Testing fees in continuous trading with two trades and no liquidity providers - testing maker fee received and maker fee paid

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    Given the parties deposit on asset's general account the following amount:
      | party         | asset | amount   |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | VEGA   | 10000000 |

    # setup recurring transfer to the maker fee reward account - this will start at the end of this epoch (1)
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                                | to_account_type                         | asset  | amount | start_epoch | end_epoch | factor |               metric                 | metric_asset | markets   |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000  | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA   | 10000  |       1     |           |    1   |  DISPATCH_METRIC_MAKER_FEES_RECEIVED |      ETH     |           |
      | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000  | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | VEGA   | 1234   |       1     |           |    2   |  DISPATCH_METRIC_MAKER_FEES_PAID     |      ETH     | ETH/DEC21 |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 10000     |
      | lpprov   | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 480    | 9520    |
      | trader3b | ETH   | ETH/DEC21 | 240    | 9760    |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
    And the accumulated infrastructure fees should be "0" for the asset "ETH"

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer    | price | size | seller  | aggressor side |
      | trader3a | 1002  | 2    | trader4 | sell           |
      | trader3b | 1002  | 1    | trader4 | sell           |

    # For trader3a-
    # trade_value_for_fee_purposes for trader3a = size_of_trade 0000000000000000000000000000000000000000000000000000000000000000 price_of_trade = 2 0000000000000000000000000000000000000000000000000000000000000000 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.002 0000000000000000000000000000000000000000000000000000000000000000 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.005 0000000000000000000000000000000000000000000000000000000000000000 2004 = 10.02 = 11 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0 0000000000000000000000000000000000000000000000000000000000000000 3006 = 0

    # For trader3b -
    # trade_value_for_fee_purposes = size_of_trade 0000000000000000000000000000000000000000000000000000000000000000 price_of_trade = 1 0000000000000000000000000000000000000000000000000000000000000000 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.002 0000000000000000000000000000000000000000000000000000000000000000 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.005 0000000000000000000000000000000000000000000000000000000000000000 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0 0000000000000000000000000000000000000000000000000000000000000000 3006 = 0

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 8      | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 302    | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25 ??
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 726    | 9285    |
      | trader3b | ETH   | ETH/DEC21 | 363    | 9643    |
      | trader4  | ETH   | ETH/DEC21 | 715    | 8958    |

    And the accumulated infrastructure fees should be "8" for the asset "ETH"
    And the accumulated liquidity fees should be "302" for the market "ETH/DEC21"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks
    # only trader3 received the maker fees so only they get the reward of 10k
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 726    | 9285    |
      | trader3b | ETH   | ETH/DEC21 | 363    | 9643    |
      | trader4  | ETH   | ETH/DEC21 | 715    | 8958    |

    Then "trader3a" should have general account balance of "6470" for asset "VEGA"
    And "trader3b" should have general account balance of "3529" for asset "VEGA"
    And "trader4" should have general account balance of "1234" for asset "VEGA"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks
    # expect no change to anyone
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 726    | 9285    |
      | trader3b | ETH   | ETH/DEC21 | 363    | 9643    |
      | trader4  | ETH   | ETH/DEC21 | 715    | 8958    |

  Scenario: Testing fees in continuous trading with two trades and one liquidity providers with 10 and 0 s liquidity fee distribution timestep - test maker fee received, taker fee paid and lp fees rewards
    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset  | amount   |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | VEGA   | 10000000 |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 10000     |

    # transfer to the maker fee received reward account and the taker paid fee reward account
    Given the parties submit the following recurring transfers:
      | id | from                                                             | from_account_type    | to                                                                | to_account_type                         | asset  | amount | start_epoch | end_epoch | factor |               metric                 | metric_asset | markets   |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000  | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA   | 10000  |       1     |           |    0.5 |  DISPATCH_METRIC_MAKER_FEES_RECEIVED |      ETH     |           |
      | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000  | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | VEGA   | 1234   |       1     |           |    1   |  DISPATCH_METRIC_MAKER_FEES_PAID     |      ETH     | ETH/DEC21 |
      | 3  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000  | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | VEGA   | 500    |       1     |           |    2   |  DISPATCH_METRIC_LP_FEES_RECEIVED    |      ETH     |           |


    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1080  | 1      |
      | buy  | 920   | 1      |
      | buy  | 910   | 210    |
      | sell | 1090  | 184    |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321    |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667    |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    # For trader3a-
    # trade_value_for_fee_purposes for trader3a = size_of_trade 0000000000000000000000000000000000000000000000000000000000000000 price_of_trade = 2 0000000000000000000000000000000000000000000000000000000000000000 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.002 0000000000000000000000000000000000000000000000000000000000000000 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.005 0000000000000000000000000000000000000000000000000000000000000000 2004 = 10.02 = 11 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.001 0000000000000000000000000000000000000000000000000000000000000000 2004 = 2.004 = 3 (rounded up to nearest whole value)

    # For trader3b -
    # trade_value_for_fee_purposes = size_of_trade 0000000000000000000000000000000000000000000000000000000000000000 price_of_trade = 1 0000000000000000000000000000000000000000000000000000000000000000 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.002 0000000000000000000000000000000000000000000000000000000000000000 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.005 0000000000000000000000000000000000000000000000000000000000000000 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] 0000000000000000000000000000000000000000000000000000000000000000 trade_value_for_fee_purposes = 0.001 0000000000000000000000000000000000000000000000000000000000000000 1002 = 1.002 = 2 (rounded up to nearest whole value)

    Then the following trades should be executed:
      | buyer    | price | size | seller  | aggressor side | buyer fee | seller fee | infrastructure fee | maker fee | liquidity fee |
      | trader3a | 1002  | 2    | trader4 | sell           | 0         | 19         | 5                  | 11        | 3             |
      | trader3b | 1002  | 1    | trader4 | sell           | 0         | 11         | 3                  | 6         | 2             |

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 8      | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 5      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general  |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321     |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667     |
      | trader4  | ETH   | ETH/DEC21 | 480    | 9490     |
      | aux1     | ETH   | ETH/DEC21 | 50978  | 99939024 |
      #| trader3a | ETH   | ETH/DEC21 | 480    | 9531     |
      #| trader3b | ETH   | ETH/DEC21 | 240    | 9766     |
      #| trader4  | ETH   | ETH/DEC21 | 679    | 9291     |

    And the accumulated infrastructure fees should be "8" for the asset "ETH"
    And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks

    # Not sure why this transfer is gone now?
    And the following transfers should happen:
      | from   | to   | from account                | to account           | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 5      | ETH   |

    # only trader3 received the maker fees so only they get the reward of 10k
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general  |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321     |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667     |
      | trader4  | ETH   | ETH/DEC21 | 480    | 9490     |
      | aux1     | ETH   | ETH/DEC21 | 50978  | 99939029 |
      #| trader3a | ETH   | ETH/DEC21 | 480    | 9531     |
      #| trader3b | ETH   | ETH/DEC21 | 240    | 9766     |
      #| trader4  | ETH   | ETH/DEC21 | 679    | 9291     |

    # 11/17 x 10000 -> maker fee received reward
    Then "trader3a" should have general account balance of "6470" for asset "VEGA"
    # 6/17 x 10000 -> maker fee recevied reward
    And "trader3b" should have general account balance of "3529" for asset "VEGA"
    # 1234 = taker fee paid reward reward
    And "trader4" should have general account balance of "1234" for asset "VEGA"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks

    # expect no change to anyone
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general  |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321     |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667     |
      | trader4  | ETH   | ETH/DEC21 | 480    | 9490     |
      | aux1     | ETH   | ETH/DEC21 | 50978  | 99939029 |
