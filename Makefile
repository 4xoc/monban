VERSION=v0.0.1-1
COMMIT=$(shell git rev-list -1 HEAD | cut -c1-10)
TIME=$(shell date -u +%a,\ %d\ %b\ %Y\ %H:%M:%S\ %Z)

all: linux freebsd darwin

linux:
	GOOS=linux GOARCH=386 go build -ldflags '-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X "main.Date=${TIME}"' -o monban.linux *.go

freebsd:
	GOOS=freebsd GOARCH=386 go build -ldflags '-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X "main.Date=${TIME}"' -o monban.freebsd *.go

darwin:
	GOOS=darwin GOARCH=386 go build -ldflags '-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X "main.Date=${TIME}"' -o monban.darwin *.go

clean:
	rm -f monban.linux || true
	rm -f monban.freebsd || true
	rm -f monban.darwin || true
