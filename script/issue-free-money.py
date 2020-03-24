#!/usr/bin/python3

import argparse
import requests
from typing import Any, Dict, List


class WalletClient(object):

    def __init__(
        self,
        url: str
    ):
        self.token = ""
        self.url = url

        self._httpsession = requests.Session()

    def _header(self) -> Dict[str, Any]:
        if self.token == "":
            raise Exception("not logged in")
        return {"Authorization": "Bearer " + self.token}

    def create(
        self,
        walletname: str,
        passphrase: str
    ) -> requests.Response:
        """
        Create a wallet using a wallet name and a passphrase. If a wallet
        already exists, the action fails. Otherwise, a JWT (json web token) is
        returned.
        """
        req = {
            "wallet": walletname,
            "passphrase": passphrase
        }
        r = self._httpsession.post(self.url + "/api/v1/wallets", json=req)
        if r.status_code != 200:
            self.token = ""
        else:
            self.token = r.json()["Data"]
        return r

    def login(
        self,
        walletname: str,
        passphrase: str
    ) -> requests.Response:
        """
        Log in to an existing wallet. If the wallet does not exist, or if the
        passphrase is incorrect, the action fails. Otherwise, a JWN (json web
        token) is returned.
        """
        req = {
            "wallet": walletname,
            "passphrase": passphrase
        }
        r = self._httpsession.post(
            "{}/api/v1/auth/token".format(self.url), json=req)
        if r.status_code == 200:
            self.token = r.json()["Data"]
        else:
            self.token = ""
        return r

    def logout(self) -> requests.Response:
        """
        Log out from a wallet. The token is deleted from the WalletClient
        object.
        """
        r = self._httpsession.delete(
            "{}/api/v1/auth/token".format(self.url), headers=self._header())
        if r.status_code == 200:
            self.token = ""
        return r

    def listkeys(self) -> requests.Response:
        return self._httpsession.get(
            "{}/api/v1/keys".format(self.url), headers=self._header())

    def generatekey(
        self,
        passphrase: str,
        metadata: List[Dict[str, str]]
    ) -> requests.Response:
        """
        Generate a new keypair with the given metadata.
        """
        req = {
            "passphrase": passphrase,
            "meta": metadata
        }
        return self._httpsession.post(
            "{}/api/v1/keys".format(self.url), json=req,
            headers=self._header())

    def signtx(self, tx, pubKey) -> requests.Response:
        """
        Sign a transaction.

        tx must be a base64-encoded string, e.g.
        tx = base64.b64encode(someBlob).decode("ascii")

        pubKey must be a hex-encoded string.
        """
        req = {
            "tx": tx,
            "pubKey": pubKey,
            "propagate": False
        }
        return self._httpsession.post(
            "{}/api/v1/messages".format(self.url), headers=self._header(),
            json=req)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Issue free money")

    parser.add_argument(
        "--wallets", type=str, required=True,
        help="comma-separated list of wallet names")

    parser.add_argument(
        "--passphrase", type=str, required=True,
        help="super-secure passphrase")

    parser.add_argument(
        "--walletserver", type=str, required=True,
        help="wallet server (e.g. https://wallet.X.vega.xyz)")

    parser.add_argument(
        "--veganode", type=str, required=True,
        help="vega node (e.g. https://geo.X.vega.xyz)")

    parser.add_argument(
        "--amount", type=str, default="10000000",
        help="amount to set balances to")

    return parser.parse_args()


def print_err(wallet: str, errstr: str, r: requests.Response):
    print("{}: {}: HTTP {} {}".format(wallet, errstr, r.status_code, r.text))


def process_wallet(
    w: WalletClient, wallet: str, passphrase: str,
    veganode: str, amount: str
):
    print("{}: Logging in to wallet server.".format(wallet))
    r = w.login(wallet, passphrase)
    if r.status_code != 200:
        if "wallet does not exist" in r.text:
            print("{}: Wallet does not exist. Creating one.".format(wallet))
            r = w.create(wallet, passphrase)
            if r.status_code != 200:
                print_err(wallet, "Failed to create wallet", r)
                return
        else:
            print_err(wallet, "Failed to log in to wallet server", r)
            return

    r = w.listkeys()
    if r.status_code != 200:
        print_err(wallet, "Failed to list keys", r)
        return

    keys = [keypair["pub"] for keypair in r.json()["Data"]]
    if len(keys) == 0:
        print("{}: Creating keypair.".format(wallet))
        r = w.generatekey(passphrase, [])
        if r.status_code != 200:
            print_err(wallet, "Failed to generate keypair", r)
            return
        keys = [r.json()["Data"]]

    for key in keys:
        req = {
            "notif": {
                "traderID": key,
                "amount": amount
            }
        }
        print("{}: Setting balances for {}".format(wallet, key))
        r = requests.post("{}/fountain".format(veganode), json=req)
        if r.status_code != 200:
            print_err(wallet, "Failed to set balances for {}".format(key), r)


def main():
    args = parse_args()
    w = WalletClient(args.walletserver)

    for wallet in args.wallets.split(","):
        process_wallet(w, wallet, args.passphrase, args.veganode, args.amount)


if __name__ == "__main__":
    main()
