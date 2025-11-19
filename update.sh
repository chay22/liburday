#!/bin/bash

gkey=$1
year=$2

if [ -z "$gkey" ]; then
  if [ -z "$GOOGLE_CALENDAR_API_KEY" ]; then
    echo "Error: Google API key is required."
    exit 1
  fi

  gkey=$GOOGLE_CALENDAR_API_KEY
fi

git clone --single-branch --branch main git@github.com:chay22/liburday.git dist
./bin/liburday-linux-amd64 --gkey $gkey --out-dir dist $year
cd dist
git add .
git commit -m "feat: update"
git push origin main
cd ..
rm -rf dist
exit 0
