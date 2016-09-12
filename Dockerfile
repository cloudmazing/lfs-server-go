FROM golang:1.5-onbuild
MAINTAINER Mike Quinn
RUN ./scripts/start
EXPOSE 8080
