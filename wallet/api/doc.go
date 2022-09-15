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
// # Third-party application workflow
//
// All applications consuming this API should start with the following workflow:
//
//  1. connect_wallet: it allows the third-party application to initiate the
//     connection with the API.
//
//  2. get_permissions: the application requires permissions from the user to
//     consume the API. As a result, it should check if it has enough.
//
//  3. request_permissions: if the application doesn't have enough permissions
//     to work, it has to request some.
package api
