ARG BUILDFLAGS
FROM --platform=$BUILDPLATFORM golang:latest AS build
WORKDIR /src/
COPY . ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "${BUILDFLAGS}" -o /bin/backend

FROM alpine:latest
RUN apk update && apk add ca-certificates shadow libcap && rm -rf /var/cache/apk/*
RUN groupadd -g 1000 backend
RUN useradd -u 1000 -g 1000 -d / backend
COPY --from=build /bin/backend /backend
RUN setcap 'cap_net_bind_service=+ep' /backend # Allow server to bind privileged ports

EXPOSE 3030
USER backend
ENTRYPOINT ["/backend"]
