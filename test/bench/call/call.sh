#!/bin/bash

# start service
sleep 2
nohup ./call_service >/dev/null 2>&1 &
pid=$!

# start edge
sleep 2
./call_edge

# kill service
kill -9 $pid