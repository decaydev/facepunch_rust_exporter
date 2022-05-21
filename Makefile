BUILD_DT:=$(shell date +%F-%T)
GO_LDFLAGS:="-s -w -extldflags \"-static\" -X main.BuildVersion=${GITHUB_REF} -X main.BuildCommitSha=${GITHUB_SHA} -X main.BuildDate=$(BUILD_DT)" 

.PHONE: build-binaries
build-binaries:
	rm -rf .build | true

	export CGO_ENABLED=0 ; \
	go -os="linux windows freebsd netbsd openbsd"        -arch="amd64 386" -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/redis_exporter-${GITHUB_REF}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	go -os="darwin solaris illumos"                      -arch="amd64"     -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/redis_exporter-${GITHUB_REF}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	go -os="linux freebsd netbsd"                        -arch="arm"       -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/redis_exporter-${GITHUB_REF}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	go -os="linux" -arch="arm64 mips64 mips64le ppc64 ppc64le s390x"       -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/redis_exporter-${GITHUB_REF}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	echo "done"
