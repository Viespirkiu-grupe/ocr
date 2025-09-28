FROM golang:1.25.1 AS builder

WORKDIR /temp

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd cmd
COPY internal internal

RUN CGO_ENABLED=0 go build -o app ./cmd/app

FROM alpine:latest
COPY --from=builder /temp/app /app

RUN apk update && apk add --no-cache tesseract-ocr tesseract-ocr-data-lit tesseract-ocr-data-eng poppler-utils ghostscript && rm -rf /var/cache/apk/*

WORKDIR /work

VOLUME [ "/inbox" ]

ENTRYPOINT ["/app"]
