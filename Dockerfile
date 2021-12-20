FROM golang:alpine as builder

ENV GOPROXY="https://goproxy.cn"

ADD . /home/s3
 
RUN cd /home/s3 && go build  -o /root/s3  cmd/s3/main.go


FROM alpine:3.14 as prod


WORKDIR /root/

COPY --from=0 /root/s3 .
RUN chmod +x /root/s3
