Feature: Fees Calculations

Scenario: Fees Calculations 

Given I have open orders on my account during cont. & auction sessions
When the order gets matched resulting in one or multple trades 
Then each fee is correctly calculated in settlement currency of the market as below
  # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes
  # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes
  # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes
  # total_fee = infrastructure_fee + maker_fee + liquidity_fee
  # trade_value_for_fee_purposes = = size_of_trade * price_of_trade
