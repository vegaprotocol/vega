## Liquidity Engine

The Liquidity Engine handles Liquidity Provisions
(`LiquidityProvisionSubmission`) and subsequent updates to the created orders.


A `LiquidityProvisionSubmission` specifies how to match a given liquidity by defining a shape of orders to be created.
These shape is updated each time there is a change in the book.
