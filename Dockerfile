FROM golang:1.7-onbuild
MAINTAINER Mike Quinn
#ENV INSTALL_PATH /go/src/github.com/cloudmazing/lfs-server-go
ENV INSTALL_PATH /go/src/lfs-server-go
RUN mkdir -p /go/src/app/lfs_content
#ENV COPY_FILE `git ls-tree --full-tree -r HEAD | awk '{print $NF}' | tr '\n' ' '`
ADD . /go/src/lfs-server-go
WORKDIR /go/src/lfs-server-go
#RUN go get github.com/tools/godep
#RUN godep go build ./...
