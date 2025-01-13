#!/bin/sh

set -eux

# This script needs to iterate all files in /in with the extension .tpl
# then run envsubst on each file and output the result to /out

for file in /in/*.tpl; do
  envsubst < $file > /out/$(basename $file .tpl)
done
