# EpochTime

This service is responsible for sending out event bus messages when Vega leaves or enters a new epoch.
It also has support for clients subscribing to updates when the epoch changes.

The length of an epoch is defined by the network parameter `ValidatorsEpochLength`

