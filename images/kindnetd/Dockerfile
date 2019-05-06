# STEP 1: Build executable binary
FROM golang:1.12 AS builder

WORKDIR /go/src/kindnet
COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -o /go/bin/kindnet ./cmd/kindnetd

# STEP 2: Build small image
FROM scratch

COPY --from=builder /go/bin/kindnet /bin/kindnet

CMD ["/bin/kindnet"]
