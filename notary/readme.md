# Notary

## What's the context for the notary

In some scenarios, the Vega network needs to be able to express a decision taken by the network (meaning in our case by all the validator nodes), in a way which can be understood by a foreign entity/foreign network (e.g ethereum).

To do so, all Vega nodes must be started with wallets recognised by these foreign entities, and which would represent the Vega node in that network. Currently *all* Vega nodes require these - in future the requirement could be changed to only include validators. These wallets are recognised by both Vega and the foreign network, which allows the node to sign messages with a signatures that can be verified by both parties.

The notary is here to keep track of all the signatures, and of the minimum amount of signatures required for the decision to be valid.
The notary also exposes a service and API which allows a user to retrieve the aggregated signatures in order to use them outside of Vega.

## Network events that require notarisation
- Approval of a [new asset](../assets/)

### Extra events coming soon
- Approval of a withdrawal
- Adding a [new validator](../validators/)

## Example use case.

In the context of submitting a [Governance](../governance/) proposal for a new erc20 token to be used in Vega, when the propopsal has been approved by the token holders, Vega needs to inform the [erc20 token bridge contract](https://github.com/vegaprotocol/MultisigControl/tree/master/contracts) that the new token needs to be enabled.

To do so, at the end of the governance process all nodes will sign a message which can be verified by the bridge contract on the Ethereum network. All of these signatures will be aggregated by the notary, which them makes them available for an end user to provide when executing the Vega Bridge contract to enable the asset to be deposited in to the system.
