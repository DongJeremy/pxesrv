NAME=pxesrv
VERSION=1.0.0
SOURCE=main.go
LDFLAGS=-ldflags "-s -w"
WLDFLAGS=-ldflags "-s -w"
BUILD_LINUX=CGO_ENABLED=0 GOARCH=amd64 GOOS=linux
BUILD_WINDOWS=CGO_ENABLED=0 GOARCH=amd64 GOOS=windows
BUILD_DARWIN=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin
OUTPUT=$(shell pwd)/output

all: dist

fmt: clean
	gofmt -l -w ./

linux: fmt
	${BUILD_LINUX} go build ${LDFLAGS} -a -o ${OUTPUT}/linux/${NAME} ${SOURCE}
	upx --brute ${OUTPUT}/linux/${NAME}

window: fmt
	${BUILD_WINDOWS} go build ${WLDFLAGS} -a -o ${OUTPUT}/window/${NAME}.exe .
	upx --brute ${OUTPUT}/window/${NAME}.exe

mac: fmt
	${BUILD_DARWIN} go build ${LDFLAGS} -a -o ${OUTPUT}/mac/${NAME} ${SOURCE}
	upx --brute ${OUTPUT}/mac/${NAME}

clean:
	rm -rf output/*

dist: linux window mac
	cp -a -f templates netboot pxe.yml ${OUTPUT}/window
	cp -a -f templates netboot pxe.yml ${OUTPUT}/linux
	cp -a -f templates netboot pxe.yml ${OUTPUT}/mac
	find ${OUTPUT} -name .gitkeep -exec rm -fr {} \;
	cd ${OUTPUT}/linux/ && zip -qr ../${NAME}-$(VERSION).linux-amd64.zip .
	cd ${OUTPUT}/window/ && zip -qr ../${NAME}-$(VERSION).window-amd64.zip .
	cd ${OUTPUT}/mac/ && zip -qr ../${NAME}-$(VERSION).darwin-amd64.zip .
