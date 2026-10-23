FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/analytics ./cmd/analytics

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/analytics /app/analytics
ENV PORT=8081
EXPOSE 8081
ENTRYPOINT ["/app/analytics"]
