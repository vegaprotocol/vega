Feature: Spot markets where mdp + pdp > adp of quote asset
 
  Background:

    Given the following assets are registered:
      | id        | decimal places |
      | BASE.6.1  | 6              |
      | QUOTE.6.1 | 6              |
    
    # Deposits to auxiliary parties
    Given the parties deposit on asset's general account the following amount:
      | party | asset     | amount     |
      | aux1  | BASE.6.1  | 1000000000 |
      | aux2  | BASE.6.1  | 1000000000 |
      | aux1  | QUOTE.6.1 | 1000000000 |
      | aux2  | QUOTE.6.1 | 1000000000 |
    # Depositis to test parties
    Given the parties deposit on asset's general account the following amount:
      | party  | asset     | amount     |
      | buyer  | BASE.6.1  | 1000000000 |
      | seller | BASE.6.1  | 1000000000 |
      | buyer  | QUOTE.6.1 | 1000000000 |
      | seller | QUOTE.6.1 | 1000000000 |

    # Create the simplest spot market with no price monitoring and no fees
    Given the spot markets:
      | id         | name       | base asset | quote asset | liquidity monitoring | risk model                | auction duration | fees         | price monitoring | sla params    | decimal places | position decimal places | tick size |
      | BASE/QUOTE | BASE/QUOTE | BASE.6.1   | QUOTE.6.1   | default-parameters   | default-simple-risk-model | 1                | default-none | default-none     | default-basic | 6              | 6                       | 1         |
    And the parties place the following orders:
      | party | market id  | side | volume | price   | resulting trades | type       | tif     | reference |
      | aux1  | BASE/QUOTE | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
      | aux2  | BASE/QUOTE | sell | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
    When the opening auction period ends for market "BASE/QUOTE"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BASE/QUOTE"
  
  
  Scenario: Trades of the smallest possible size (price set so notional can be represented in asset decimals)

    Given the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0     |
      | market.fee.factors.infrastructureFee | 0     |
    And the liquidity fee factor should be "0" for the market "BASE/QUOTE"

    Given the parties place the following orders:
      | party  | market id  | side | volume | price   | resulting trades | type       | tif     | reference |
      | buyer  | BASE/QUOTE | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
      | seller | BASE/QUOTE | sell | 1      | 1000000 | 1                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
    When the following trades should be executed:
      | buyer | size | price   | seller | buyer fee | seller fee |
      | buyer | 1    | 1000000 | seller | 0         | 0          |
    Then the parties should have the following account balances:
      | party | asset     | general    |
      | buyer | BASE.6.1  | 1000000001 |
      | buyer | QUOTE.6.1 | 999999999  |
    And the parties should have the following account balances:
      | party  | asset     | general    |
      | seller | BASE.6.1  | 999999999  |
      | seller | QUOTE.6.1 | 1000000001 |


  Scenario: Trades of the smallest possible price (size set so notional can be represented in asset decimals)

    Given the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0     |
      | market.fee.factors.infrastructureFee | 0     |
    And the liquidity fee factor should be "0" for the market "BASE/QUOTE"

    Given the parties place the following orders:
      | party  | market id  | side | volume  | price | resulting trades | type       | tif     | reference |
      | buyer  | BASE/QUOTE | buy  | 1000000 | 1     | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
      | seller | BASE/QUOTE | sell | 1000000 | 1     | 1                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
    When the following trades should be executed:
      | buyer | size    | price | seller | buyer fee | seller fee |
      | buyer | 1000000 | 1     | seller | 0         | 0          |
    Then the parties should have the following account balances:
      | party | asset     | general    |
      | buyer | BASE.6.1  | 1001000000 |
      | buyer | QUOTE.6.1 | 999999999  |
    And the parties should have the following account balances:
      | party  | asset     | general    |
      | seller | BASE.6.1  | 999000000  |
      | seller | QUOTE.6.1 | 1000000001 |


  Scenario: Trades of the smallest possible size and price

    Given the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0     |
      | market.fee.factors.infrastructureFee | 0     |
    And the liquidity fee factor should be "0" for the market "BASE/QUOTE"

    Given the parties place the following orders:
      | party  | market id  | side | volume | price | resulting trades | type       | tif     | reference |
      | buyer  | BASE/QUOTE | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | p3b2-1    |
      | seller | BASE/QUOTE | sell | 1      | 1     | 1                | TYPE_LIMIT | TIF_GTC | p3b1-1    |
    When the following trades should be executed:
      | buyer | size | price | seller | buyer fee | seller fee |
      | buyer | 1    | 1     | seller | 0         | 0          |

    # ISSUE: None of the quote asset can be represented in asset decimals therefore none exhanged.
    Then the parties should have the following account balances:
      | party | asset     | general    |
      | buyer | BASE.6.1  | 1000000001 |
      | buyer | QUOTE.6.1 | 1000000000 |
    And the parties should have the following account balances:
      | party  | asset     | general    |
      | seller | BASE.6.1  | 999999999  |
      | seller | QUOTE.6.1 | 1000000000 |
