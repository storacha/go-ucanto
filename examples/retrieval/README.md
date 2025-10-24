# Retrieval Server Example

This example demonstrates how a retrieval server works. You can use the client to fetch the following hashes:

* `zQmWvQxTqbG2Z9HPJgG57jjwR154cKhbtJenbyYTWkjgF3e`
* `zQmY7Bpsk9Qvorkx1R47bnFwWFtTTVanL1gELhq31siJVJT`
* `zQmTpxKkDpsHrEKwSVN4WteuWPmHehswVRww7zvyfotopzo`

## Getting started

Start the server:

```sh
go run ./server
```

Retrieve some data:

```sh
go run ./client zQmWvQxTqbG2Z9HPJgG57jjwR154cKhbtJenbyYTWkjgF3e
```

You can make a byte range request like:

```sh
go run ./client zQmWvQxTqbG2Z9HPJgG57jjwR154cKhbtJenbyYTWkjgF3e 0 4
```
