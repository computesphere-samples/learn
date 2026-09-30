# Built by the Publish lab images workflow with this directory as the context.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=$(cat VERSION)" -o /out/app . \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
# /data belongs to the app's user, so the note can be saved with or without a
# volume mounted there.
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
