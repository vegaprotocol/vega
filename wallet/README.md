Wallet
======

This package provide the basic cryptography to sign the vega transaction, and a basic key management system: `wallet service`.

A wallet takes the form of a file saved in the filesystem and encrypted using a passphrase of the choice of the user.
A wallet is composed of a list of keypairs (ed25519) used to signed transaction for the user of the wallet.

# wallet service

## generate configuration

The package provide a way to generate the configuration of the service before starting it, it can be used through the vega command line like so:
```
$ vega wallet service init --genrsakey -f
```
Where --genrsakey will generate for you rsa key used to sign the jwt tokens, and -f will rewrite an existing configuration if one is found.
This command will generate the new configuration inside your vega root folder.

You can then start your vega service like so:
```
$ vega wallet service run
```

## Available functionnalities

### Create a wallet
Creation of a wallet is done using a wallet name and a passphrase.
If a wallet already exists, this action is aborted, if not, a new wallet is created, marshalled, encrypted using the passphrase and saved in a file in on the fileystem.
A session and jwt token is then created, and the jwt token is returned to the user.

### login to a wallet
Login to a wallet is done using a wallet name and a passphrase.
If the wallet do not exist, the action is aborted, if the wallet exists, then we try to decrypt it using the passphrase, if decryption failed, this is aborted.
If decryption succeed, the wallet is loaded, a session is created, and a jwt token is returned.

### logout from a wallet.
Using a valid jwt token, the session is recovered, and removed from the service. The wallet is then not accessible anymore using the token.

### list keys
Using a jwt token, a user can list all his public keys (with taint status, metadata). First the token is validated, then the session is extracted from it, then the wallet
informations are recovered and sent back to the user.

### generate a new keypair
Using a valid jwt token and a passphrase, we recover the session of the users, then we try to open the wallet of the user using the passphrase, if one of these action did not succeed
and error will be returned. Then a new keypair is generated for the user, and saved in the wallet. The new public key is then returned to the user.
