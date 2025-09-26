FROM golang:alpine AS builder

WORKDIR /temp

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 go build -o app ./cmd/app

FROM alpine:latest
COPY --from=builder /temp/app /app

RUN apk update && apk add --no-cache tesseract-ocr tesseract-ocr-data-lit tesseract-ocr-data-eng poppler-utils ghostscript && rm -rf /var/cache/apk/*

WORKDIR /

VOLUME [ "/inbox" ]

ENTRYPOINT ["/app"]