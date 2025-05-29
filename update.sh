#!/bin/bash

gkey=$1
year=$2

if [ -z "$gkey" ]; then
  echo "Error: Google API key is required."
  exit 1
fi

git clone git@github.com:chay22/liburday.git dist
./bin/liburday-linux-amd64 --gkey $gkey --out-dir dist $year
cd dist
git add . 
git commit -m "feat: update"
git push origin main
cd ..
rm -rf dist
exit 0