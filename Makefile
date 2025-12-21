PKGROOT=github.com/Ladicle/kubectl-check
OUTDIR=dist

.PHONY: build test clean

build:
	goreleaser build --snapshot --clean

test:
	go fmt $(PKGROOT)/...
	go vet $(PKGROOT)/...
	go test $(PKGROOT)/...

clean:
	-rm -r $(OUTDIR)
