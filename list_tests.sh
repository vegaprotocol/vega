#!/bin/bash -e
	echo -e "==STARTED==="
	for f in $(find * -name '*.feature'); 
	do 
		outcome=$(go test  -v ./integration/ --godog.format=progress "$(pwd)/$f" | tail -1 | head -c 4); 
		if [[ $outcome == *"FAIL"* ]]; then
		echo -e "\033[0;31m$f"
		fi
	done
	echo -e "============"
	echo -e "==FINISHED=="
	echo -e "============"