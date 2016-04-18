#!/bin/bash

set -ue
dir="$1"

for r in 1000 5000 10000 20000 50000
do
    for t in plain ltsv json apache
    do
        echo "`date +%Y-%m-%d-%H:%M:%S`: benchmark $t $r/sec"
        rm -f "$dir/bench.$t.log"
        sleep 1
        go-dummer-simple -r "$r" -s 1 -i $t.log -o "$dir/bench.$t.log"
        echo "`date +%Y-%m-%d-%H:%M:%S`: done. wrote `wc -l $dir/bench.$t.log` lines"
        sleep 5
    done
done
