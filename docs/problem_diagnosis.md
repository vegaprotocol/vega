There are a few tools, scripts and commands that we use to start investigating a problem while working on Vega. This document collects a few of those.

# Tools worth knowing about.
## Vega specific tools
* [Block explorer](https://explorer.vega.trading/) ([repo](https://github.com/vegaprotocol/explorer)), which contains [an API for decoding blocks](https://github.com/vegaprotocol/explorer#api).
* [vegastream](https://github.com/vegaprotocol/vega/tree/develop/cmd/vegastream)

## API Specific tools
### GraphQL
* [GraphQURL](https://github.com/hasura/graphqurl) is a curl-ish CLI tool that supports subscriptions
* [GraphQL Playground](https://github.com/prisma-labs/graphql-playground) is served by Vega nodes serving a GraphQL API
* Simplest of all: CURL can be used to post queries.

### GRPC
### REST

# Some hypothetical situations

<details>
  <summary><strong>Is [insert network] down?</strong></summary>

  The quickest check is [stats.vega.trading](https://stats.vega.trading) ([repo](https://github.com/vegaprotocol/stats/)). You should see the network there, and most or all of the stats rows should have a green block, implying it's healthy.
  
  Stats is a really simply web view of the REST [statistics endpoint](https://docs.testnet.vega.xyz/api/rest/#operation/Statistics), so you could also use curl. Choose a node serving REST from this [devops repo document](https://github.com/vegaprotocol/devops-infra/blob/master/doc/vega_environments.md) and then curl the statistics endpoint:
  ```bash
  curl https://n04.d.vega.xyz/statistics
  ```
  
  If this fails, totally it could be that the node itself is down, while the network is fine. If you get a 502 error, then the machine is up, the HTTPS proxy is working, but the Vega node is not running.

  If you want to skip Vega and see if Tendermint is healthy, you can try going straight to Tendermint's RPC port. Choose a node that exposes the Tendermint RPC from this [devops repo document](https://github.com/vegaprotocol/devops-infra/blob/master/doc/vega_environments.md) and then fetch the status endpoint:
  ```bash
  curl https://n01.d.vega.xyz/tm/status
  ```

 If those two fail, you can try SSHing to the machine to see what's up. The [devops repo](https://github.com/vegaprotocol/devops-infra/blob/master/doc/vega_environments.md) will list all of the nodes, and how you can connect to them to investigate further.
</details
