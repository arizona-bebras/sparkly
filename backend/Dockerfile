FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/backend ./cmd/backend

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/backend /app/backend
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/backend"]
