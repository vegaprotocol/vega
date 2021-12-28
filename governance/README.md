## Governance engine

The governance engine receives proposals that have passed through consensus (meaning they come from the chain). For each block, the `OnChainTimeUpdate` function is called, which will have the engine check currently active proposals to see if the voting period for these proposals has expired.
Once the voting period has expired, the engine will check to see if the proposal was accepted or rejected. Rejected proposals, obviously, have reached the end of their useful lives. Accepted ones, however, are fed back into the execution engine to be enacted.

Votes will also be fed into this engine. Votes are expected to be cast on proposals by ID (a vote can only be cast on a proposal once it's considered valid). We track all votes (yes/no votes).

## Dependencies

The Governance engine needs to be able to check the accounts of parties (seeing how many tokens a given party has, and how many total tokens are in the system). This information is currently accessible through the collateral engine.

To expose the proposals API (gRPC), we'll create a buffer which will allow the governance service to stream updates regarding proposals over a gRPC call, should this be required.

## Accepted proposals

Accepted proposals are returned from the `OnChainTimeUpdate` call, for the execution engine to act on them.

## Convenience

For the governance service to more easily return the requested data (e.g. `GetProposalByReference`), there are some convenience functions available to expose active proposals by reference and id.