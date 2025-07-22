FROM golang:1.22 as builder
ARG GIT_TOKEN

WORKDIR /workspace

COPY ./ .

RUN git config --global url."https://${GIT_TOKEN}:x-oauth-basic@github.com/coredgeio".insteadOf "https://github.com/coredgeio"

RUN go env -w GOPRIVATE=github.com/coredgeio/*

RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /workspace/workflow-manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/workflow-manager .
COPY default.yml /opt/config.yml
USER 65532:65532

ENTRYPOINT ["/workflow-manager"]
