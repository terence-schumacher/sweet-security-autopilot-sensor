#!/bin/bash

API_KEY="9a5b7137-7281-43ad-9152-c30b03b75ee2"
API_SECRET="0b37701c-bcbe-40e2-8965-e939527a826b"

JSON_PAYLOAD='{"architecture": "'$(uname -m)'"'

if [[ -n "$HTTP_PROXY" ]]; then
JSON_PAYLOAD+=',"httpProxy": "'"$HTTP_PROXY"'"'
fi

if [[ -n "$HTTPS_PROXY" ]]; then
JSON_PAYLOAD+=',"httpsProxy": "'"$HTTPS_PROXY"'"'
fi

JSON_PAYLOAD+='}'

curl -fLs -X POST -o /tmp/install.sh "https://control.sweet.security/v1/update/install/script" -H "X-Api-Key: ${API_KEY}" -H "X-Api-Secret: ${API_SECRET}" -H "Content-Type: application/json" -d "$JSON_PAYLOAD"

chmod +x /tmp/install.sh
/tmp/install.sh

rm -f /tmp/install.sh