Feature: Multiple successor markets allowed for single parent

  Background:

    # Create some assets
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount                     |
      | lpprov | ETH   | 10000000000000000000000000 |
      | aux1   | ETH   | 10000000000000000000000000 |
      | aux2   | ETH   | 10000000000000000000000000 |

    # Create some oracles
    ## Oracle for parent
    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec19Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec19Oracle" is given in "5" decimal places
    ## Oracle for successors
    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "5" decimal places


  Scenario: Two pending successor markets, and one proposed successor market. A pending successor market is enacted and the remaining successors are rejected (0081-SUCM-014)

    # Create a parent market, two pending successor markets, and one proposed successor market
    Given the markets:
      | id         | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | is passed |
      | ETH/DEC19  | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle     | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   | true      |
      | ETH/DEC20a | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle     | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                | true      |
      | ETH/DEC20b | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle     | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                | true      |
      | ETH/DEC20c | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle     | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                | false     |
    Then the market state should be "STATE_PENDING" for the market "ETH/DEC19"
    And the market state should be "STATE_PENDING" for the market "ETH/DEC20a"
    And the market state should be "STATE_PENDING" for the market "ETH/DEC20b"
    And the market state should be "STATE_PROPOSED" for the market "ETH/DEC20c"

    # LP submits commitments to the parent and all three successor markets. The funds are transfered to the relevant bond accounts.
    Given the parties submit the following liquidity provision:
      | id  | party  | market id  | commitment amount       | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19  | 10000000000000000000000 | 0.3 | buy  | BID              | 1          | 1      | submission |
      | lp1 | lpprov | ETH/DEC19  | 10000000000000000000000 | 0.3 | sell | ASK              | 1          | 1      | submission |
      | lp2 | lpprov | ETH/DEC20a | 10000000000000000000000 | 0.3 | buy  | BID              | 1          | 1      | submission |
      | lp2 | lpprov | ETH/DEC20a | 10000000000000000000000 | 0.3 | sell | ASK              | 1          | 1      | submission |
      | lp3 | lpprov | ETH/DEC20b | 10000000000000000000000 | 0.3 | buy  | BID              | 1          | 1      | submission |
      | lp3 | lpprov | ETH/DEC20b | 10000000000000000000000 | 0.3 | sell | ASK              | 1          | 1      | submission |
      | lp4 | lpprov | ETH/DEC20c | 10000000000000000000000 | 0.3 | buy  | BID              | 1          | 1      | submission |
      | lp4 | lpprov | ETH/DEC20c | 10000000000000000000000 | 0.3 | sell | ASK              | 1          | 1      | submission |
    And the parties should have the following account balances:
      | party  | asset | market id  | general                   | margin | bond                    |
      | lpprov | ETH   | ETH/DEC19  | 9960000000000000000000000 | 0      | 10000000000000000000000 |
      | lpprov | ETH   | ETH/DEC20a | 9960000000000000000000000 | 0      | 10000000000000000000000 |
      | lpprov | ETH   | ETH/DEC20b | 9960000000000000000000000 | 0      | 10000000000000000000000 |
      | lpprov | ETH   | ETH/DEC20c | 9960000000000000000000000 | 0      | 10000000000000000000000 |

    # ETH/DEC20a is the first successor to exit the opening auction (becomes enacted)
    When the parties place the following orders:
      | party | market id  | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC20a | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC20a | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20a | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20a | sell | 10     | 15    | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC20a"

    # Check ETH/DEC20a is enacted, ETH/DEC20b and ETH/DEC20ac are rejected, and the relevant bond accounts are emptied
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC20a"
    And the last market state should be "STATE_REJECTED" for the market "ETH/DEC20b"
    And the last market state should be "STATE_REJECTED" for the market "ETH/DEC20c"
    And the parties should have the following account balances:
      | party  | asset | market id  | general                   | margin                 | bond                    |
      | lpprov | ETH   | ETH/DEC19  | 9978421146481368239000000 | 0                      | 10000000000000000000000 |
      | lpprov | ETH   | ETH/DEC20a | 9978421146481368239000000 | 1578853518631761000000 | 10000000000000000000000 |
      | lpprov | ETH   | ETH/DEC20b | 9978421146481368239000000 | 0                      | 0                       |
      | lpprov | ETH   | ETH/DEC20c | 9978421146481368239000000 | 0                      | 0                       |







    

