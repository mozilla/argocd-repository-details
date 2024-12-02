FROM golang:1.23 AS builder

WORKDIR /app

COPY reference-api/ ./

RUN go mod download
RUN go get .

COPY . .

RUN mkdir -p build
RUN CGO_ENABLED=0 go build -v -o build ./...

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/build/reference-api /app/reference-api

EXPOSE 8000

CMD ["/app/reference-api"]
