FROM golang:1.22-alpine AS builder
WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && go build -trimpath -ldflags="-s -w" -o /out/gin-looklook ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
ENV TZ=Asia/Shanghai
USER app
COPY --from=builder /out/gin-looklook /usr/local/bin/gin-looklook
EXPOSE 8080 4000
ENTRYPOINT ["gin-looklook"]
