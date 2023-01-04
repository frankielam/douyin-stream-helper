FROM golang:alpine as builder
WORKDIR /app 
    
COPY ./main.go /app
# RUN go build -o main ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -ldflags="-w -s" -o main main.go

FROM alpine:edge
RUN set -xe && \
    apk add --update --no-cache ffmpeg && \
    apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata \  
    && mkdir -p /app/out

WORKDIR /app 
   
COPY --from=builder /app/main /app
COPY ./Arial.ttf /app

CMD ["/app/main"]