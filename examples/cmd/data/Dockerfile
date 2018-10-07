# We use a multi-stage build, so we require Docker >=17.05 to build these
# example images
FROM golang:alpine as builder
RUN go version
RUN apk add git
COPY . /go/src/github.com/jaypipes/gsr/examples/data
WORKDIR /go/src/github.com/jaypipes/gsr/examples/data
RUN set -x && \
    go get github.com/golang/dep/cmd/dep && \
    dep ensure -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

# Take the built binary from the builder image and place it into a new
# from-scratch image, reducing the resulting image size substantially
FROM scratch
COPY --from=builder /go/src/github.com/jaypipes/gsr/examples/data/main /app/
WORKDIR /app
CMD ["./main"]
