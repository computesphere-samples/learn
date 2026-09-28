# Built by a Quay build trigger with this directory as the context.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=$(cat VERSION)" -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
