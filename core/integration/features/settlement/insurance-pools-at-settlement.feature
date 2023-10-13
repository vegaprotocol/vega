Feature: Test the transfers to and from the insurance pools when markets terminate in various ways
  Background:
    Given the average block duration is "1"
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |

    # Max pegged orders 2, min auction duration 1s, constant MTM, 5s successor window length
    And the following network parameters are set:
      | name                                         | value |
      | market.auction.minimumDuration               | 1     |
      | market.auction.maximumDuration               | 10    |
      | network.markPriceUpdateMaximumFrequency      | 0s    |
      | market.liquidity.successorLaunchWindowLength | 5s    |
      | limits.markets.maxPeggedOrders               | 2     |
    
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999999  | 300               |
    And the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.02               |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec19Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE2" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE2" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    # Create 4 markets, all with the same settlement asset, different configs, because we can...
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | lognormal-risk-model-fish     | default-margin-calculator | 1                | default-none  | default-none       | ethDec19Oracle         | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC20 | ETH        | ETH   | default-log-normal-risk-model | margin-calculator-1       | 1                | default-none  | default-none       | ethDec20Oracle         | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC21 | ETH        | ETH   | default-simple-risk-model     | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle         | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC22 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
      | ETH/DEC23 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
    # set their insurance pool to some balance
    And the initial insurance pool balance is "1000000" for all the markets

  @InsurancePools
  Scenario: Check insurance pool balances when markets are settled in a variety of ways, at different points in time
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount        |
      | party1  | ETH   | 1000000000000 |
      | party2  | ETH   | 1000000000000 |
      | party3  | ETH   | 1000000000000 |
      | party4  | ETH   | 1000000000000 |
      | party5  | ETH   | 1000000000000 |
      | party6  | ETH   | 1000000000000 |
      | aux1    | ETH   | 1000000000000 |
      | aux2    | ETH   | 1000000000000 |
      | lpprov1 | ETH   | 1000000000000 |
      | lpprov2 | ETH   | 1000000000000 |
      | lpprov3 | ETH   | 1000000000000 |
      | lpprov4 | ETH   | 1000000000000 |
    # Provide liquidity on 19, 20, 21
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 90000000          | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 90000000          | 0.1 | submission |
      | lp3 | lpprov3 | ETH/DEC21 | 90000000          | 0.1 | submission |
      | lp3 | lpprov3 | ETH/DEC21 | 90000000          | 0.1 | submission |
      | lp4 | lpprov4 | ETH/DEC22 | 90000000          | 0.1 | submission |
      | lp4 | lpprov4 | ETH/DEC22 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party   | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov2 | ETH/DEC20 | 2         | 1                    | buy  | BID              | 500    | 10     |
      | lpprov2 | ETH/DEC20 | 2         | 1                    | sell | ASK              | 500    | 10     |
      | lpprov3 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 500    | 10     |
      | lpprov3 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 500    | 10     |
      | lpprov4 | ETH/DEC22 | 2         | 1                    | buy  | BID              | 500    | 10     |
      | lpprov4 | ETH/DEC22 | 2         | 1                    | sell | ASK              | 500    | 10     |

    # ETH/DEC19 leaves opening auction
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | lpprov1 | ETH/DEC19 | buy  | 225    | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | lpprov1 | ETH/DEC19 | sell | 36     | 250   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC23"

    ## Now ETH/DEC20 leaves opening auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the mark price should be "150" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC23"

    ## let's have ETH/DEC21 leave opening auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC21 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC21 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the mark price should be "150" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC23"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC21"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC22"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC23"

    ## final market to leave auction is ETH/DEC22
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party5 | ETH/DEC22 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/DEC22 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/DEC22 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | party6 | ETH/DEC22 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the mark price should be "150" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC23"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC21"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC22"
    And the insurance pool balance should be "1000000" for the market "ETH/DEC23"
    ## Now cancel the first market, before it even left opening auction...
    # No succession to consider, the insurance pool is instantly distributed
    When the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC23 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 150              |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC23"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC21"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
	And the global insurance pool balance should be "200000" for the asset "ETH"

    # OK, let's terminate a market via governance that is in continuous trading
    # The successor time window now comes in to play
    When the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC22 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 150              |
    Then the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC23"
    # insurance pool is not yet distributed
    And the insurance pool balance should be "1200000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC21"
    And the insurance pool balance should be "1200000" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
	And the global insurance pool balance should be "200000" for the asset "ETH"
    # pass the successor time window
    When the network moves ahead "10" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    # these markets no longer exist, so we can ignore them
    #And the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC22"
    #And the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC23"
    ## now we should see the update
    And the insurance pool balance should be "1500000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC21"
    ## Nothing in the drained pool, nothing goes to the insurance pool of the old market
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
	And the global insurance pool balance should be "500000" for the asset "ETH"

    ## Now settle a market via the oracle
    When the oracles broadcast data signed with "0xCAFECAFE2":
      | name               | value |
      | trading.terminated | true  |
    And the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC21"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC21"
    ## Nothing in the drained pool, nothing goes to the insurance pool of the old market
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
	And the global insurance pool balance should be "500000" for the asset "ETH"
    # Moving past the successor window means nothing here, market is not settled
    When the network moves ahead "10" blocks
    Then the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC21"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC20"
    And the insurance pool balance should be "1500000" for the market "ETH/DEC21"
    ## Nothing in the drained pool, nothing goes to the insurance pool of the old market
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
	And the global insurance pool balance should be "500000" for the asset "ETH"

    ## Now settle the terminated market
    When the oracles broadcast data signed with "0xCAFECAFE2":
      | name             | value |
      | prices.ETH.value | 150   |
    And the network moves ahead "10" blocks
    Then the insurance pool balance should be "2000000" for the market "ETH/DEC19"
    And the insurance pool balance should be "2000000" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    ## Nothing in the drained pool, nothing goes to the insurance pool of the old market
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
	And the global insurance pool balance should be "1000000" for the asset "ETH"

    ## Now terminate both of the other markets
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
    And the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    Then the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_NO_TRADING" for the market "ETH/DEC19"
    And the insurance pool balance should be "2000000" for the market "ETH/DEC19"
    And the insurance pool balance should be "2000000" for the market "ETH/DEC20"
	And the global insurance pool balance should be "1000000" for the asset "ETH"

    ## Now settle one market, pass the successor window and ensure the insurance pool is divided between the global insurance pool and the terminated market
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 150   |
    And the network moves ahead "10" blocks
    Then the insurance pool balance should be "3000000" for the market "ETH/DEC19"
	And the global insurance pool balance should be "2000000" for the asset "ETH"
    ## the insurance pools from the settled/cancelled markets are all drained
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"

    ## Now settle the last market, and ensure the insurance pool balance is fully transferred to the global pool
    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 150   |
    And the network moves ahead "10" blocks
	And the global insurance pool balance should be "5000000" for the asset "ETH"
    ## the insurance pools from the settled/cancelled markets are all drained
    Then the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
