# STEP 1: Build executable binary
FROM golang:1.12 AS builder
ENV GO111MODULE=on

WORKDIR /go/src/kindnet
COPY . .
RUN go get -d -v ./...
RUN GOOS=linux CGO_ENABLED=0 go build -o /go/bin/kindnet ./cmd/kindnetd

# STEP 2: Build small image
FROM scratch

COPY --from=builder /go/bin/kindnet /bin/kindnet

CMD ["/bin/kindnet"]