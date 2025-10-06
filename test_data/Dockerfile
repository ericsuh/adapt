FROM golang:1.25 AS builder

WORKDIR /app
COPY . .
RUN go build .

FROM ubuntu:24.04 AS tester

WORKDIR /app
COPY test_data/Aptfile .
COPY --from=builder /app/adapt .

ENTRYPOINT [ "/app/adapt" ]
