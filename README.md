# OSSSH - OpenStack SSH

Use the magic of NetworkNamespaces and the local MetadataPort to port-forward a tcp-port to VM without a floating IP address.

## Issues

- Netnsproxy compiled against glibc

## Run

```bash
$ osssh -u myusername $uuid
```

## Build
```bash
$ go build -o osssh cmd/osssh/osssh.go
```
