# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

COPY *.go ./

RUN go build -v -o /prusa_metrics_handler

FROM alpine:latest

COPY --from=builder /prusa_metrics_handler .

ENTRYPOINT ["/prusa_metrics_handler"]