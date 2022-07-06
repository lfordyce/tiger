#!/bin/bash

build() {
  docker build -t tiger-db:latest ./src
}

run() {
  docker run -d --name timescaledb -p 5432:5432 -e POSTGRES_PASSWORD=password tiger-db:latest
}

help() {
  echo "  possible arguments:"
  echo "    build"
  echo "    run"
}

if [ -z "$1" ]; then
  echo "no argument specified"
  help
  exit 1
fi

if [ $1 = "build" ]; then
  build
elif [ $1 = "run" ]; then
  run
else
  echo "invalid argument"
  help
  exit 1
fi