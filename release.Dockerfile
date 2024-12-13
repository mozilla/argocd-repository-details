FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY reference-api .
USER 65532:65532
EXPOSE 8000

CMD ["/app/reference-api"]