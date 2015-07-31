LFS Server Go\!
======

[lfs]: https://github.com/github/git-lfs
[api]: https://github.com/github/git-lfs/blob/master/docs/api.md

LFS Server Go\! is an example server that implements the [Git LFS API][api]. 

This is based off of lfs-test-server.  There are a few differences though.  

1. This server uses the same base directory for all objects
1. The backing store is offloaded to 
  * Redis
  * BoltDB
  * Cassandra
    * TODO: password auth for cassandra
1. There is a notion of project -\> OID membership, which is lacking from the original.  This is wired up but not yet functional. It will shortly allow for validating a user's membership to a project and the project's associated OID to the user, thus ensuring a user's access to a project will allow for access to an OID
1. It will (not yet) allow for object store storage of objects

## Installing

Alternatively, use the Go installer:

```
  $ go install github.com/memikequinn/lfs-server-go
```

## Building

To build from source, use the Go tools:

```
  $ go get github.com/memikequinn/lfs-server-go
```


## Running

<b> Set your GO_ENV. Options are `prod` or `dev` or `test`</b>

Running the binary will start an LFS server on `localhost:8080` by default.
All of the configuration settings are stored in config.ini.
> You'll want to copy config.ini.example to config.ini

To use the LFS test server with the Git LFS client, configure it in the repository's `.gitconfig` file:


```
  [lfs]
    url = "http://localhost:8080/janedoe/lfsrepo"

```

HTTPS:

NOTE: If using https with a self signed cert also disable cert checking in the client repo.

```
	[lfs]
		url = "https://localhost:8080/jimdoe/lfsrepo"

	[http]
		sslfverify = false

```


An example usage:


Generate a key pair
```
openssl req -x509 -sha256 -nodes -days 2100 -newkey rsa:2048 -keyout mine.key -out mine.crt
```

Make yourself a run script

Update the config to point at your new keys

Build the server

```
go build

```

Run

```
./scripts/start.sh

```

