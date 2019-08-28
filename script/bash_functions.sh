#!/bin/bash

# This file should be sourced, not run.

json_escape() {
	echo -n "$1" | python -c 'import json,sys; print(json.dumps(sys.stdin.read()))'
}

slack_notify() {
	slack_hook_url="${SLACK_HOOK_URL:-}"
	if test -z "$slack_hook_url" ; then
		return
	fi
	if ! which curl 1>/dev/null ; then
		return
	fi
	channel="${1:-tradingcore-notify}"
	icon_emoji="${2:-:thinking-face:}"
	text="${3:-A slack notification}"
	# Escape text in preparation for inclusion in JSON payload
	channel="$(json_escape "$channel")"
	icon_emoji="$(json_escape "$icon_emoji")"
	text="$(json_escape "$text")"
	curl -XPOST --silent -H 'Content-Type: application/json' \
		--data '{"channel": '"$channel"', "icon_emoji": '"$icon_emoji"', "text": '"$text"', "username": "Autodeploy"}' \
		"$slack_hook_url" >/dev/null
}
