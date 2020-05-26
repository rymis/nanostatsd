#!/bin/sh

# Convert any data to go array
if [ "x$1" = "x" ]; then
    echo "Error: usage $0 filename"
    exit 1
fi

NAME=`md5sum "$1" | awk '{print $1}'`
FILENAME=`echo -n "$1" | sed 's/^\.\///g'`
MYNAME=`basename "$0"`
if [ "x$MYNAME" = "x$FILENAME" ]; then
    # Do not add myself
    exit 0
fi

echo "// FILE: $1"
echo "var x_$NAME *staticWebPage = newStaticWebPage(\`$FILENAME\`, \`"
base64 < "$1"
echo "\`)"

