#!/bin/bash -e

function gen_code() {
	buf generate
	if buf generate; then
		exit $?
	fi
}

gen_code
