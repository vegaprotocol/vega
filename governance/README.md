## Governance engine

The governance engine receives proposals that have passed through consensus (meaning they come from the chain). For each block, the `OnChainTimeUpdate` function is called, which will have the engine check currently active proposals to see if the voting period for these proposals has expired.
Once the voting period has expired, the engine will check to see if the proposal was accepted or rejected. Rejected proposals, obviously, have reached the end of their useful lives. Accepted ones, however, are fed back into the execution engine to be enacted.

Votes will also be fed into this engine. Votes are expected to be cast on proposals by ID (a vote can only be cast on a proposal once it's considered valid). We track all votes (yes/no votes).

## Dependencies

The Governance engine needs to be able to check the accounts of parties (seeing how many tokens a given party has, and how many total tokens are in the system). This information is currently accessible through the collateral engine.

To expose the proposals the API (gRPC), we'll create a buffer which will allow the governace service to stream updates regarding proposals over a gRPC call, should this be required.

## Accepted proposals

Accepted proposals are returned from the `OnChainTimeUpdate` call, for the execution engine to act on them.

## Convenience

For the governance service to more easily return the requested data (e.g. `GetProposalByReference`), there are some convenience functions available to expose active proposals by reference and id.

## Network Parameters
- [NetworkParams.go](./networkparams.go)

contains information on the required network parameters and their current configuration


## Engine
* [type Engine](./engine.go#L83-L87)

The actual governance engine. As any engine, it embeds the governance config, has access to the dependencies and a logger. The mutex is used for config updates, `currentTime` is the current block time, `net` represents the network params for governance.

The 2 maps `proposals` and `proposalRefs` hold the active proposals by ID (`proposals`), and by reference (`proposalRefs`). Both hold pointer values to ensure they point to the same object. We use both maps to prevent proposals with duplicate references/ID's to be submitted, so we don't accidentally overwrite an existing proposal.

### ProposalData
* [ProposalData](./engine.go#L83-L87)

This is the governance domain object representing a proposal. In the world of gRPC, a proposal and a vote are distinct messages. As far as governance is concerned, a proposal has a one-to-many relation with votes. We store yes and no votes in corresponding `map[string]*types.Vote`s.
