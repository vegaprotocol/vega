Wallet
======

This package provides the basic cryptography to sign vega transactions, and a basic key management system: `wallet service`.

A wallet takes the form of a file saved on the file system and is encrypted using the passphrase chosen by the user.
A wallet is composed of a list of key pairs (Ed25519) used to sign transactions for the user of a wallet.

# wallet service

## generate configuration

The package provides a way to generate the configuration of the service before starting it, it can be used through the vega command line like so:

```shell
vega wallet service init --genrsakey -f
```

Where `--genrsakey` generates an RSA key that will be used to sign your JWT (JSON web token), and `-f` overwrites any existing configuration files (if found).
In short: this command will generate the RSA key, and the configuration files required by the wallet service.

You can then start your vega service like so:
```shell
vega wallet service run
```

## Available functionalities

### Create a wallet
Creating a wallet is done using a name and passphrase. If a wallet already exists, the action is aborted. New wallets are marshalled, encrypted (using the passphrase) and saved to a file on the file system.
A session and accompanying JWT is created, and the JWT is returned to the user.

### logging in to a wallet
Logging in to a wallet is done using the wallet name and passphrase.
The operation fails should the wallet not exist, or if the passphrase used is incorrect (i.e. the passphrase cannot be used to decrypt the wallet).
On success, the wallet is loaded, a session is created and a JWT is returned to the user.

### logging out from a wallet.
Using the JWT returned when logging in, the session is recovered and removed from the service. The wallet can from this point on no longer be accessed using the token.

### list keys
Users can list all their public keys (with taint status, and metadata), if they provide the correct JWT. The service extracts the session from this token, and uses it to fetch the relevant wallet information to send back to the user.

### generate a new key pair
The user can has to submit a valid JWT, and a passphrase. We recover the session of the user, and attempt to open the wallet using the passphrase. If the JWT is invalid, the session could not be recovered, or the wallet could not be opened, an error is returned.
If all went well, a new key pair is generated, saved in the wallet, and the public key is returned.
