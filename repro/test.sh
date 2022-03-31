#!/bin/sh
set -euo pipefail

readonly root="$(dirname $0)"
cd "$root"

readonly oldvers=v0.34.15
readonly newvers=v0.35.x
readonly addr=localhost:26657
readonly tmhome="$PWD/tmhome"

mkdir -p bin

install() {
    local vers="$1"
    set -x; trap 'set +x' RETURN
    GOBIN=$PWD/bin go install github.com/tendermint/tendermint/cmd/tendermint@"$vers"
    mv bin/tendermint bin/tendermint-"$vers"
}

transaction() {
    local key="$1"
    local val="$2"
    diag ":: adding transaction $key = $val"
    curl --fail-with-body -s "http://$addr/broadcast_tx_commit?tx=\"${key}=${val}\"" \
	    | jq -r .result.hash
}

diag() { echo "-- $@" 1>&2; }

diag "Installing Tendermint CLI"
for vers in "$oldvers" "$newvers" ; do
    diag ":: version $vers"
    install "$vers"
done

diag "Starting TM $oldvers"
rm -fr -- "$tmhome"
./bin/tendermint-"$oldvers" --home="$tmhome" init
./bin/tendermint-"$oldvers" --home="$tmhome" start \
		 --proxy_app=kvstore \
		 --consensus.create_empty_blocks=0 2>/dev/null 1>&2 &
sleep 2

hash1="$(transaction t1 alpha)"
diag ":: transaction hash is $hash1"
hash2="$(transaction t2 bravo)"
diag ":: transaction hash is $hash2"
hash3="$(transaction t3 charlie)"
diag ":: transaction hash is $hash3"

sleep 5
diag "Stopping TM $oldvers"
kill %1; wait

diag "Editing [fastsync] to [blocksync]"
sed -i'' -e '/^\[fastsync\]$/c\
[blocksync]' "$tmhome/config/config.toml"

diag "Migrating databases with $newvers"
./bin/tendermint-"$newvers" --home="$tmhome" key-migrate

diag "Starting TM $newvers"
./bin/tendermint-"$newvers" --home="$tmhome" start \
		 --proxy-app=kvstore \
		 --consensus.create-empty-blocks=0 &
sleep 2

kill %1; wait
