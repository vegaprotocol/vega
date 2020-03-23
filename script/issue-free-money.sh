#!/bin/bash

syntax="Syntax: $0 https://walletserver/api/v1 'passphrase' https://veganode 123400000"

# wallet server
ws="${1:-}"
passphrase="${2:-}"
# vega node
node="${3:-}"
amount="${4:-}"

if test -z "$ws" -o -z "$passphrase" -o -z "$node" -o -z "$amount"; then
	echo "$syntax"
	exit 1
fi

for u in $VEGANET_USERS
do
	echo "$u: Logging in to wallet server."
	token="$(curl -s -XPOST -d '{"wallet":"'"$u"'","passphrase":"'"$passphrase"'"}' "$ws/auth/token" | jq -r .Data)"
	if test "$token" == null ; then
		echo "$u: Wallet does not exist, creating one."
		token="$(curl -s -XPOST -d '{"wallet":"'"$u"'","passphrase":"'"$passphrase"'"}' "$ws/wallets" | jq -r .Data)"
		if test "$token" == null ; then
			echo "$u: Failed to create wallet. Skipping."
			continue
		fi
	fi
	hdr="Authorization: Bearer $token"
	echo "$u: Getting a list of public keys."
	keys="$(curl -s -XGET -H "$hdr" "$ws/keys" | jq -r '.Data|.[]|.pub')"
	if test -z "$keys" ; then
		echo "$u: Creating keypair."
		keys="$(curl -s -XPOST -H "$hdr" -d '{"passphrase":"'"$passphrase"'","meta":[]}' "$ws/keys" | jq -r .Data)"
	fi
	echo "$u: User has $(echo "$keys" | wc -l) keypairs."
	for key in $keys ; do
		echo "$u: Getting free money for $key."
		curl -s -XPOST -d '{"notif":{"traderID":"'"$u"'","amount":"'"$amount"'"}}' "$node/fountain" 1>/dev/null
	done
done
