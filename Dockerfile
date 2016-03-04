FROM golang:1.5-onbuild
MAINTAINER Mike Quinn
RUN ./scripts/start.sh
EXPOSE 8080
