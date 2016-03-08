LFS Server Go\!
======
# NOT yet ready for primetime and is going through some iterations, specifically regarding access control.

[lfs]: https://github.com/github/git-lfs
[api]: https://github.com/github/git-lfs/blob/master/docs/api.md

LFS Server Go\! a server that implements the [Git LFS API](https://github.com/github/git-lfs/tree/master/docs/api) 

This is based off of [lfs-test-server](https://github.com/github/lfs-test-server)  

1. This server provides access to 
1. The meta store is offloaded to 
  * BoltDB
  * Cassandra
1. There is a notion of project -\> OID membership, which is lacking from the original.  This is wired up but still a WIP. It will allow for validating a user's membership to a project and the project's associated OID to the user, thus ensuring a user's access to a project will allow for access to an OID

##TODO:
1. Update access
  * ~~Rename user to namespace~~
  * Implement namespace and project based access
1. Remove/Clean up old objects on delete
1. When an object is public and AWS is enabled, offload GETs directly to AWS
1. Adopt [verification of uploads](https://github.com/github/git-lfs/tree/master/docs/api#verification)
1. Redo the UI so it is abstracted into its own app  

## Installing

Use the Go installer, this will install all dependencies tracked:

```
  $ go install github.com/cloudmazing/lfs-server-go
```

To use a specific config file, set `LFS_SERVER_GO_CONFIG=/path/to/config`

Then start with `./scripts/start.sh`

## Building

To build from source, use the Go tools + godep:

```
  $ go get github.com/cloudmazing/lfs-server-go
  $ go install ./...
  $ godep restore
```

## Making changes

To build from source, use the Go tools:

```
  $ go get github.com/cloudmazing/lfs-server-go
  $ go install ./...
  <edit files>
  $ godep save ./...
  $ godep update ./...
```

## Running

<b> Set your GO\_ENV. Options are `prod` or `dev` or `test`</b>

Running the binary will start an LFS server on `localhost:8080` by default.
All of the configuration settings are stored in config.ini.
> You'll want to copy config.ini.example to config.ini

### An example usage:

Generate a key pair
```
openssl req -x509 -sha256 -nodes -days 2100 -newkey rsa:2048 -keyout mine.key -out mine.crt
```

Update the config to point at your new keys

Build the server

```
go build
```

### Start it

```
./scripts/start.sh
```

## Client 
### Further client documentation on the client is available at https://git-lfs.github.com/

To use the LFS test server with the Git LFS client, configure it in the repository's `.gitconfig` file:

```
  [lfs]
    url = "http://localhost:8080/janedoe/lfsrepo"
```

This file _MUST_ be checked into git inside of your project.

HTTPS:

NOTE: If using https with a self signed cert also disable cert checking in the client repo.

```
	[lfs]
		url = "https://localhost:8080/jimdoe/lfsrepo"

	[http]
		sslfverify = false
```

## Security Design

Namespaces -\> projects
Users are given access to a namespace: read, write, or both
Users are given access to a project: read, write, or both


