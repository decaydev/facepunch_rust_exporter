BUILD_DT:=$(shell date +%F-%T)
GO_LDFLAGS:="-s -w -extldflags \"-static\" -X main.BuildVersion=${VERSION} -X main.BuildCommitSha=${GITHUB_SHA} -X main.BuildDate=$(BUILD_DT)" 

.PHONE: build
build:
	rm -rf .build | true

	export CGO_ENABLED=0 ; \
	gox -os="linux windows freebsd netbsd openbsd" -arch="amd64 386" -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/facepunch_rust_exporter-${VERSION}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	gox -os="darwin solaris"                       -arch="amd64"     -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/facepunch_rust_exporter-${VERSION}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	gox -os="linux"                                -arch="arm arm64" -verbose -rebuild -ldflags $(GO_LDFLAGS) -output ".build/facepunch_rust_exporter-${VERSION}.{{.OS}}-{{.Arch}}/{{.Dir}}" && \
	echo "done"
