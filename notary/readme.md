# Notary

## What's the context for the notary

In some scenario, the vega network needs to be able to express a decision taken by the network (meaning in our case by all the validator nodes), in a way which can be understood by a foreign entity/foreign network (e.g ethereum).
To do so, all vega node must be started with wallets recognised by these foreign entities, and which would represent the vega node in the network, using these wallets, which are recognise by both vega and the foreign network, the node can then sign messages, the signatures can then be verified by both parties.
The notary is here to keep track of all the signatures, and of the minimum amount of signatures required for the decision to be valid.
The notary also expose a service and api which allow a user to retrieve the aggregated signatures in order to use them outside of vega.

## Example of use case.

In the context of submitting through governance a new erc20 token to be used into vega, when all the governance process have finished, and the asset have been vote to be enabled into vega, it's required for vega, to communicate to the erc20 token bridge that the new token needs to be unable.
To do so at the end of the governance process, all nodes will sign a messge which can be verified by the bridge on the ethereum network, all these signature will be aggregated by the notary, which them makes them available for the client, to use them when executing smart contracts of the vega bridge on ethereum.
