#!/bin/sh
set -e
cd "$(dirname "$0")"
python3 - <<'PY'
h=open('web/head.html').read(); a=open('web/app.js').read().replace('__RENDERER__',open('web/renderer.js').read())
open('web/index.html','w').write(h+'<script>\n'+a+'\n</script></body></html>\n')
PY
GOFLAGS=-mod=mod GOPROXY=off go vet .
GOOS=windows GOARCH=amd64 GOFLAGS=-mod=mod GOPROXY=off go build -trimpath -ldflags "-s -w -H=windowsgui" -o dist/DEMviewer.exe .
GOFLAGS=-mod=mod GOPROXY=off go build -o dist/demviewer-linux .
mkdir -p dist/maps && cp maps/*.bsp.gz dist/maps/
