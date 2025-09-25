FROM golang:1.24 AS builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /zc_rss_parser

FROM alpine
COPY --from=builder /zc_rss_parser /zc_rss_parser
EXPOSE 8080

# Run
CMD ["/zc_rss_parser"]
