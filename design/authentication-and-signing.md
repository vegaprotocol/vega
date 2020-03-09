Authentication & Signing
========================

Note: this documentation does not cover the authentication provided by the soon-to-be-deprecated [auth package](../auth/). It also will allude to the [Wallet service](../wallet/README.md), which *can* be used to provide basic key management for users of your node, but is not required for signing.

## Workflow
- Vega will soon require that all transactions are signed by a keypair
- To help developers iterate quickly, the core Vega node exposes a number of endpoints that can be used to generate an unsigned protocol buffer representing a transaction
- This protocol buffer must be signed with a keypair, and then submitted to the node using the `submitTransaction` endpoint

### Wallet service
The [Wallet service](../wallet/README.md) can be run either locally or on a shared node to reduce the number of roundtrips in this transaction. See the following diagrams to understand the flow of signing an API call using the wallet service

#### Best case
In the best case, a client can generate and sign protobuffs locally:

```mermaid
sequenceDiagram
    participant Vega Node
    participant Client

	Client->>Client: Generate unsigned protobuff 
	Client->>Client: Sign protobuff 
	Client->>Vega Node: submitTransaction 
```					

#### Worst case
In the worst case, a client that is *not* running an auth service locally, and cannot generate its own protobuff representation for an order will need to make multiple calls between the Wallet Service and the Vega Node: 

```mermaid
sequenceDiagram
    participant Vega Node
    participant Client
    participant Wallet Service

	Client->>Wallet Service: Log in
	Wallet Service->>Client: Auth token 
	Client->>Client: Create REST/GraphQL call 
	Client->>Vega Node: Use appropriate prepareX endpoint
	Vega Node->>Vega Node: Generate unsigned protobuff 
	Vega Node->>Client: Return unsigned order protobuff 
	Client->>Wallet Service: Sign order protobuff
	Wallet Service->>Wallet Service: Sign protobuff 
	Wallet Service->>Client: Signed protobuff 
	Client->>Vega Node: submitTransaction 
```					

As you can see, this leaves a lot of room for variation. A node may be able to sign, but for some reason (perhaps the language has no protobuff library) may not be able to produce its own protobuffers. In this case it can leverage the prepare endpoints in the node, but sign the call.

There is one more optimisation in this chain: the wallet service can be instructed to sign & forward to the node in the same call, skipping another round trip:

```mermaid
sequenceDiagram
    participant Vega Node
    participant Client
    participant Wallet Service

	Client->>Client: Generate unsigned protobuff
	Client->>Wallet Service: Sign ond forward protobuff
	Wallet Service->>Wallet Service: Sign protobuff 
	Wallet Service->>Vega Node: Signed protobuff 
```					
