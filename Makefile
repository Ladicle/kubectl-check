PKGROOT=github.com/Ladicle/kubectl-diagnose
OUTDIR=dist

.PHONY: build check clean

build:
	goreleaser build

check:
	go vet $(PKGROOT)/...
	go test $(PKGROOT)/...

clean:
	-rm -r $(OUTDIR)
