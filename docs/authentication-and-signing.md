Authentication & Signing
========================

Note: this documentation does not cover the authentication provided by the soon-to-be-deprecated [auth package](../auth/).

## Workflow
- Vega will soon require that all transactions are signed by a keypair
- As a convenience to client developers each Vega node exposes a number of endpoints that can be used to generate an unsigned protocol buffer representing a transaction
- This protocol buffer must be signed with a keypair, and then submitted to the node using the `submitTransaction` endpoint
- [vega wallet](../wallet/README.md) is a service that can be run to manage your keys

The basic interaction between services is shown below:

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
	Client->>Wallet Service: Sign protobuff
	Wallet Service->>Wallet Service: Sign protobuff
	Wallet Service->>Client: Signed protobuff
	Client->>Vega Node: Submit signed transaction
```

This is a lot of communication between services, so we have provided various shortcuts. The interactions below will all result in a valid, signed order being submitted on the user's behalf.

### Clients can generate their own protobuffs
The `prepareX` endpoints (e.g.) `prepareOrder` endpoints are provided **for clients that do not have a protobuff library**. This may be true for developers hacking on bots or tools using the REST or GraphQL API who may not want to add the weight of a protobuff library to their code. If you are using a GRPC client already, or are happy to bring in a library for your platform, you can skip this step and go straight to requesting the wallet service sign your transactions:

```mermaid
sequenceDiagram
    participant Vega Node
    participant Client
    participant Wallet Service

	Client->>Wallet Service: Log in
	Wallet Service->>Client: Auth token
	Client->>Client: Generate unsigned order
	Client->>Wallet Service: Sign order
	Wallet Service->>Wallet Service: Sign order
	Wallet Service->>Client: Signed protobuff
	Client->>Vega Node: Submit signed transaction
```

### `vega wallet` can sign *and submit* in the same request
By providing an extra parameter to the wallet service, it can automatically submit the signed transaction to an API node on your behalf. This saves the client from having to submit it manually.

```mermaid
sequenceDiagram
    participant Vega Node
    participant Client
    participant Wallet Service

	Client->>Client: Generate unsigned protobuff
	Client->>Wallet Service: Sign and forward protobuff
	Wallet Service->>Wallet Service: Sign protobuff
	Wallet Service->>Vega Node: Submit signed transaction
```

### Clients can sign their own transactions
The functionality provided by the wallet service can be integrated in to your client. `vega` uses Ed25519 - visit the [wallet](../wallet/README.md) folder to find out more.

```mermaid
sequenceDiagram
    participant Vega Node
    participant Client

	Client->>Client: Generate signed protobuff
	Client->>Vega Node: Signed protobuff
```


## Hosting wallet service for multiple users
The [Wallet service](../wallet/README.md) should be run locally, allowing you to access your keypairs for signing transactions. `vega wallet` can also be hosted centrally to manage the keys of multiple users, but this brings with it security considerations that are outside the scope of this document.

