PKGROOT=github.com/Ladicle/kubectl-diagnose
OUTDIR=dist

.PHONY: build check clean

build:
	goreleaser build --snapshot --rm-dist

check:
	go vet $(PKGROOT)/...
	go test $(PKGROOT)/...

clean:
	-rm -r $(OUTDIR)
