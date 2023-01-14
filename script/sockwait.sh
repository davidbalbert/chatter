#!/bin/sh

while [ ! -S /tmp/ospfd.sock ]; do
    sleep 1
done
