Authentication & Signing
========================

Note: this documentation does not cover the authentication provided by the soon-to-be-deprecated [auth package](../auth/). It also will allude to the [Wallet service](../wallet/README.md), which *can* be used to provide basic key management for users of your node, but is not required for signing.

## Workflow
- Vega will soon require that all transactions are signed by a keypair
- To help developers iterate quickly, the core Vega node exposes a number of endpoints that can be used to generate an unsigned protocol buffer representing a transaction
- This protocol buffer must be signed with a keypair, and then submitted to the node using the `submitTransaction` endpoint

### Wallet service
The [Wallet service](../wallet/README.md) can be run either locally or on a shared node to reduce the number of roundtrips in this transaction.
