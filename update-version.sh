#!/bin/bash

ver=$(git log -n 1 --pretty=oneline --format=%D | awk -F, '{print $1}' | awk '{print $3}')
if [ "$ver" = "master" ]
then
    ver="master ($(git log -n 1 --pretty=oneline --format=%h))"
fi
sed -i "s/^const version = .*$/const version = \"$ver\"/" cmd/gq-server/version.go
sed -i "s/^const version = .*$/const version = \"$ver\"/" cmd/gq-client/version.go
