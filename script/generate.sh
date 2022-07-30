#!/bin/bash

function gen_code() {
	buf generate
	if [ $? -ne 0 ]; then
		exit $?
	fi
}

gen_code
