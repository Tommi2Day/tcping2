#!/bin/bash
if golangci-lint \
	run ./... \
	--timeout=5m \
	--out-format colored-line-number \
	--exclude-dirs-use-default; then
	echo "OK"
else
	echo "FAIL"
fi

