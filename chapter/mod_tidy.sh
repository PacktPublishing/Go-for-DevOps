#!/bin/bash

# This will find every directory with a go.mod file and run go mod tidy on it.

root=`pwd`

for d in `find . -type d -print`; do
	cd "$root/$d";

	found=false;
	for file in `find . -type f -name 'go.mod' -maxdepth 1 -print`; do
		found=true;
		break;
	done;

        if $found
        then
                echo "Has go.mod file: $d";
		go mod tidy
        fi
done;
