VERSION = v0.0.1
COMMIT = $(shell git rev-list -1 HEAD | cut -c1-10)
TIME = $(shell date -u +%a,\ %d\ %b\ %Y\ %H:%M:%S\ %Z)
PLATFORMS = darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm linux/arm64 netbsd/386 netbsd/amd64 netbsd/arm openbsd/386 openbsd/amd64 openbsd/arm solaris/amd64

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

default:
	go build -ldflags '-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X "main.Date=${TIME}"' -o monban *.go

release: $(PLATFORMS) checksums

$(PLATFORMS):
	GOOS=$(os) GOARCH=$(arch) go build \
    -ldflags '-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X "main.Date=${TIME}"' \
    -o monban-${VERSION}.$(os)-$(arch) *.go && \
    tar -zcf monban-${VERSION}.$(os)-$(arch).tar.gz monban-${VERSION}.$(os)-$(arch)

checksums:
	sha256sum *tar.gz > sha256sums.txt

clean:
	rm -f *.tar.gz || true
	rm -f monban-* || true
	rm -f monban || true
	rm -f sha256sums.txt || true

playground:
	bash playground_scripts/create_playground.sh

playground-clean:
	docker rm -f my-openldap-container

test:
	go test -coverprofile=coverage.out
