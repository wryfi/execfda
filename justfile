MODULE := "gitlab.com/wryfi/rwx"
VERSION := `git describe --tags --dirty 2> /dev/null || echo v0`
REVISION := `git rev-parse --short HEAD 2> /dev/null || echo 0`
BUILD_DATE := `date -u +'%FT%T%:z'`

LDFLAG_VERSION := "-X main.Version=" + VERSION
LDFLAG_REVISION := "-X main.Revision=" + REVISION
LDFLAG_BUILD_DATE := "-X main.BuildDate=" + BUILD_DATE

LDFLAGS :=  LDFLAG_VERSION + " " + LDFLAG_REVISION + " " + LDFLAG_BUILD_DATE + " -w -s"

default:
    @just --list --justfile {{ justfile() }}

build:
    GOARCH=arm64 GOOS=darwin go build -o build/rwx_{{ VERSION }}_darwin_arm64 -ldflags "{{ LDFLAGS }}" .

run:
    go run -ldflags "{{ LDFLAGS }}" .
