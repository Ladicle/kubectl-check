FROM gcr.io/distroless/static-debian10@sha256:73c9f3f9ee72a6070499e46826dacfda8cb428cb97bef9a70ee6f67957ec99c6
COPY kubectl-check /opt/bin/
ENTRYPOINT ["/opt/bin/kubectl-check"]
