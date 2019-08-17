ifneq (,${GO_FLAGS})
	GFLAGS=${GO_FLAGS}
else
	GFLAGS=-ldflags "-s -w"
endif

all:
	GOPATH=`pwd` GOARCH=amd64 go build Gotp

install:amd64 arm arm64 mips64 mips64le ppc64 ppc64le win32 win64

amd64:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
arm:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
arm64:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
mips64:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
mips64le:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
ppc64:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
ppc64le:
	GOPATH=`pwd` GOARCH=$@ go build $(GFLAGS) -o Gotp_$@ Gotp
win64:
	GOPATH=`pwd` GOOS=windows GOARCH=amd64 go build $(GFLAGS) -o Gotp.exe Gotp
win32:
	GOPATH=`pwd` GOOS=windows GOARCH=386 go build $(GFLAGS) -o Gotp32.exe Gotp
	
