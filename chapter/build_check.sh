#!/bin/bash

# Running this file will recursively dive into every directory that has a .go file
# and run "go test" in that directory. 
# Note: this will fail on the k8 stuff, as it has a complicated test setup.

root=`pwd`

for d in `find . -type d -print`; do
	#echo "found $d";
	cd "$root/$d";

	goDir=false;
	for file in `find . -type f -name '*.go' -maxdepth 1 -print`; do
		goDir=true;
		break;
	done;

        if $goDir
        then
                echo "Has Go files: $d";
		#go build -o /tmp/
		go test
        fi
done;

#go build -o /tmp/
