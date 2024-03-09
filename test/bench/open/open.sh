#!/bin/bash

# start service
sleep 2
nohup ./open_service >/dev/null 2>&1 &
pid=$!

# start edge
sleep 2
./open_edge

# kill service
kill -9 $pid