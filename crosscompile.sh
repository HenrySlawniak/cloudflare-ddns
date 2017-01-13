#!/bin/bash

mkdir -p bin/

go get -v

for targetArch in amd64 386 arm arm64 mips64 mips64le ppc64 ppc64le; do
  for targetOS in windows linux darwin freebsd openbsd netbsd plan9 solaris dragonfly android; do
    echo "Compiling $targetOS:$targetArch"
    export GOOS=$targetOS
    export GOARCH=$targetArch

    OUT=bin/$(basename $(echo $PWD))_${GOOS}_${GOARCH}
    if [ $GOOS == "windows" ]
    then
      OUT="$OUT.exe"
    fi
    bash -c "go build -ldflags '-w' -o $OUT ."
  done
done
