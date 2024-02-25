#!/bin/bash

# start service
sleep 2
nohup ./publish_service >/dev/null 2>&1 &
pid=$!

# start edge
sleep 2
./publish_edge

# kill service
kill -9 $pid