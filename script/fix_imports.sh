#!/bin/bash

# Fix a validator.pb.go file by making the import order deterministic.

if ! test -f "$1" ; then
	echo "Syntax: fix_imports.sh some.validator.pb.go"
	exit 1
fi

awkscript="$(mktemp)"
cat >"$awkscript" <<EOF
# AWK script to remove all blank lines in the "imports" section of a golang source file.
BEGIN {
	IMP=0
}
IMP==1 && /^\)$/ {
	IMP=0
}
(IMP==1 && !/^$/) || IMP==0 {
	print
}
IMP==0 && /^import \($/ {
	IMP=1
}
EOF

f="$(mktemp)" && \
awk -f "$awkscript" <"$1" >"$f" && \
rm -f "$awkscript" && \
mv "$f" "$1" && \
goimports -w "$1"
