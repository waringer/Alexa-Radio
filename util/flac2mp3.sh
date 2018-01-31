#!/usr/local/bin/bash

echo Content-type: audio/mpeg
echo ""

fp=`/usr/bin/printenv ALIAS`
doc=`/usr/bin/printenv DOCUMENT_URI`

/usr/local/bin/flac -csd "$fp$doc" | /usr/local/bin/lame --silent -V0 -
