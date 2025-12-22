#!/bin/bash
if golangci-lint \
	run ./... \
	--timeout=5m ; then
	echo "OK"
else
	echo "FAIL"
fi

