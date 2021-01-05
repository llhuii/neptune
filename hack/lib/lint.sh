#!/usr/bin/env bash

neptune::lint::check() {
    cd ${NEPTUNE_ROOT}
    echo "start lint ..."
    echo "check any issue by golangci-lint ..."
    GOOS="linux" golangci-lint run -v -c
}
