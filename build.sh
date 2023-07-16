export CGO_ENABLED=1
export CC=aarch64-linux-musl-gcc
export GOARCH=arm64
export GOHOSTARCH=amd64
export EXTRA_LDFLAGS='-extldflags=-fuse-ld=bfd -extld=aarch64-linux-musl-gcc'
go build -ldflags="${EXTRA_LDFLAGS}" -buildmode=plugin -o xebula.so .
