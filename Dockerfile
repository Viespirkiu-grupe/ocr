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

RUN apk update && apk add --no-cache tesseract-ocr tesseract-ocr-data-lit tesseract-ocr-data-eng poppler-utils ghostscript mupdf-tools
RUN apk add build-base wget tar
RUN wget http://foremost.sourceforge.net/pkg/foremost-1.5.7.tar.gz -O /tmp/foremost.tar.gz && \
    tar -xvzf /tmp/foremost.tar.gz -C /tmp
WORKDIR /tmp/foremost-1.5.7
RUN sed -i 's/fopen64/fopen/g' *.c && \
    sed -i 's/fseeko64/fseeko/g' *.c && \
    sed -i 's/ftello64/ftello/g' *.c && \
    sed -i 's/fstat64/fstat/g' *.c && \
    sed -i 's/stat64/stat/g' *.c && \
    sed -i 's/lseek64/lseek/g' *.c
RUN sed -i 's/RAW_FLAGS = -Wall -O2/RAW_FLAGS = -Wall -O2 -fcommon/g' Makefile
RUN mkdir -p /usr/share/man/man8 && make && make install
RUN rm -rf /tmp/foremost-1.5.7 /tmp/foremost.tar.gz /var/cache/apk/*
WORKDIR /work

VOLUME [ "/inbox" ]

ENTRYPOINT ["/app"]
