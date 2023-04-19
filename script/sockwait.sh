#!/bin/sh

while [ ! -S "$1" ]; do
    sleep 1
done
