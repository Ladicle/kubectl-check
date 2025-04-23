PKGROOT=github.com/Ladicle/kubectl-check
OUTDIR=dist

.PHONY: build check clean

build:
	goreleaser build --snapshot --rm-dist

check:
	go fmt $(PKGROOT)/...
	go vet $(PKGROOT)/...
	go test $(PKGROOT)/...

clean:
	-rm -r $(OUTDIR)
