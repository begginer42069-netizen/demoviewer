#!/bin/sh
# Builds DEMviewer for Windows and Linux into dist/.
# VERSION (e.g. 1.2.0) is shown in the app; it defaults to "dev".
set -e
cd "$(dirname "$0")"
VERSION="${VERSION:-dev}"
python3 - <<'PY'
h=open('web/head.html').read(); a=open('web/app.js').read().replace('__RENDERER__',open('web/renderer.js').read()).replace('__STUDIO__',open('web/studio.js').read())
open('web/index.html','w').write(h+'<script>\n'+a+'\n</script></body></html>\n')
PY
export GOFLAGS=-mod=mod GOPROXY=off
go vet .
mkdir -p dist
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui -X main.version=$VERSION" -o dist/DEMviewer.exe .
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o dist/DEMviewer-linux-amd64 .
# extra test maps, only if a local maps/ folder exists (not part of the repo)
if ls maps/*.bsp.gz >/dev/null 2>&1; then mkdir -p dist/maps && cp maps/*.bsp.gz dist/maps/; fi
echo "built DEMviewer $VERSION"
