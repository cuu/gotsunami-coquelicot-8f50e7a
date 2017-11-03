#!/bin/sh

# Writes a version file with the latest git commit id
# and any tag associated with it.


cat > version.go << EOF
package main

const (
    appVersion = "gotsunami-coquelicot-8f50e7a"
)
EOF

go build && rm -f version.go
