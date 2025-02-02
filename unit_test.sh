#!/bin/bash

TEST_RESULT=0
DOCKERFILE='Dockerfile-test'
IMAGE='changelog'
TEARDOWN_RAN=0

teardown() {

    [ "$TEARDOWN_RAN" -ne "0" ] && return

    echo "Running teardown..."

    docker rm -f "$TEST_CONTAINER_NAME"
    TEARDOWN_RAN=1
}

trap teardown EXIT ERR SIGINT SIGTERM

mkdir -p artifacts

get_N_chars_commit_hash() {

    local CHARS=${1:-7}

    git rev-parse --short="$CHARS" HEAD
}

TEST_CONTAINER_NAME="changelog-$(get_N_chars_commit_hash 7)"

echo "Building image"
docker build -f "$DOCKERFILE" -t "$IMAGE" .

echo -e "\n---------------------------------------------------------------\n"

echo "Running container"
docker run -d --rm --name "$TEST_CONTAINER_NAME" "$IMAGE" sleep infinity

echo -e "\n---------------------------------------------------------------\n"

echo "Installing dependencies"
docker exec --workdir /workdir "$TEST_CONTAINER_NAME" make install > 'artifacts/install_logs.txt'

echo -e "\n---------------------------------------------------------------\n"
echo "Running tests"
docker exec --workdir /workdir -e PATH=/opt/app-root/src/go/bin:$PATH "$TEST_CONTAINER_NAME" make test > 'artifacts/test_logs.txt'
TEST_RESULT=$?

cat artifacts/test_logs.txt

echo -e "\n---------------------------------------------------------------\n"

if [ $TEST_RESULT -eq 0 ]; then
    echo "Tests ran successfully"
else
    echo "Tests failed..."
    sh "exit 1"
fi
