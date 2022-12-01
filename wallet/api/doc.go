// Package api exposes all the wallet capabilities to manage connections,
// wallets, keys, and transactions.
//
// It is built with th JSON-RPC v2 standard. More info:
// See https://www.jsonrpc.org/specification for more information.
//
// # Terminology
//
// User:
//
//	The user is the actor behind the API that has the responsibility to
//	review and validate the requests. It can be a human or a bot.
//
// Wallet front-end:
//
//	The wallet front-end is the interface through which the user interact
//	with the API. It can be a command-line interface, a graphical
//	user-interface, or a script.
//
// Third-party application:
//
//	The application that connects to the API (through HTTP or directly to
//	JSON-RPC) to access wallets information, send transaction, etc.
//
// # Third-party application connection workflow
//
// Before sending transaction, a third-party application must initiate a connection.
// The connection workflow consists of:
//
//  1. Obtaining a connection token.
//
//  2. Verifying if the state of the permissions given to the connection.
//
//  3. Request additional permissions if needed.
//
// ## 1. Obtaining a connection token
//
// A connection token is required to access the protected methods.
//
// There are two type of tokens, and the way to obtain them differs:
// - Temporary tokens
// - Long-living tokens
//
// Once obtained, this token must be used to value the `token` property in the
// requests payload of protected endpoints.
//
// ### Temporary tokens
//
// A temporary token is unique, randomly generated and is only valid for the
// duration of the connection between the third-party application and the wallet
// service.
//
// If any of the third-party application or the wallet service decide to end
// the connection, the token is voided. It can't be reused. There is no need for
// third-party applications to save this token for future use.
//
// This token is issued when the third-party application explicitly request a
// connection to a wallet through the `client.connect_wallet` endpoint.
//
// To end the connection, the third-party application can call the `client.disconnect_wallet`.
// endpoint. Once called, the token can no longer be used.
//
// Explicitly requiring a connection requires manual intervention from the user
// to review to connection. While this type of token useful for auditing the
// connections, and make them short-living, it's not suited for a headless software
// (like bots, simulator, etc.) that requires full automation and complete
// independence. For that type of software, long-living-tokens are better suited.
//
// ### Long-living tokens
//
// A long-living token, just like a temporary one, is unique and randomly generated.
// However, it's not voided when the connection between the third-party application,
// and the wallet service ends, nor when the service shutdowns.
//
// Akin to a traditional API token, it can be reused. This aspect makes is
// particularly useful for a headless software to operate independently.
//
// A long-living token must be generated in advance, before the service starts.
// through the `admin.generate_service_token` endpoint. Once generated, and the
// service started, the headless application can skip the call to the `client.connect_wallet`
// endpoint, and directly start calling the protected endpoints.
//
// It's not possible to close a connection initiated with a long-living token.
// Calling the `client.disconnect_wallet`.
//
// ## 2. Verifying the permissions
//
// To ensure a fine-grained control on what a third-party application can or
// can't do, the users can associate a set of permissions to the third-party
// application connection.
//
// In order to figure out what's the state of the permission, the third-party
// application must call the `client.get_permissions` endpoint and compare the
// result with what it requires. It's up to the third-party application to
// determine what it needs access to, and to which extent.
//
// If it has enough permission, then it can proceed with the call to other
// protected methods. If it doesn't, it has to request additional ones.
//
// ## 3. Requesting permissions
//
// If, for any reason, the third-party application doesn't have enough permissions
// to do its job, it has to request addition ones through the `client.request_permissions`
// endpoint.
package api
