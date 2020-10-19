banking
=======

This package provide an engine which is an abstraction on top of the Collateral engined and the External Resource checker.

One method will be provided for each type of ChainEvent dealing with Collateral and for each asset supported, as of now:
- Asset_Allowlisted
- Asset_Deposited
- Asset_Withdrawn

Once one of these methods called, the banking will setup into the external resource checker some validation to be done,
e.g: Asset_Deposited validation for an erc20 token requires to look at the eth event logs to confirm that the deposit
really did happen.

After the validation is confirmed, the engine will finalize the processing of the ChainEvent by calling the appropriate
function on the Collateral engine.
