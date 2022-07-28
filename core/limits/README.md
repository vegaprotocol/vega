Network limits
==============

This package allow the configuration of network wide limits / restriction. This restriction are set as part of the genesis
block and will be valid for the whole duration of the network. The only way to update them would be to start a brand new
network with a new set of these settings.

Here's the list of the settings available:
- `propose_market_enabled`: type=boolean,  are markets proposal allowed
- `propose_asset_enabled`: type=boolean, are assets proposal allowed
- `propose_market_enabled_from`: type=date, optional, from when markets proposal allowed
- `propose_asset_enabled_from`: type=date, optional, from when assets proposal allowed

All dates are to be specified in the RFC3339 format, any invalid date would cause the genesis state to be invalid
therefore the network would stop straight away.

For each setting, the boolean value have the priority to the date, this means that if both a boolean value and date are specified
but also the boolean value is false, then the given setting will never be enabled.

Example settings:
```json
{
	"app_state": {
		"network_limits": {
			"propose_market_enabled": true, // market proposal enabled
			"popose_asset_enabled": false, // asset proposal disabled forever
			"propose_market_enabled_from": "2021-12-31T23:59:59Z" // this is in UTC timezone, market proposal will be enabled at this date
			// propose_asset_enabled_from is omitted
		}
	}
}
```
