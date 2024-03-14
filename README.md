# OSSSH - OpenStack SSH

Use the magic of NetworkNamespaces and the local MetadataPort to port-forward a tcp-port to VM without a floating IP address.

## ⚠ Warning

The code is probably written very poorly as it is my first golang project. It's barely working and should be considered as a PoC.

## Issues

- Cannot create listener sometimes
- Netnsproxy will not stop on premature exit
- Netnsproxy compiled against glibc
- cleanup Routine needs to be refactored

## Run

```bash
$ osssh -u myusername $uuid
```

## Build
```bash
$ go build -o osssh cmd/osssh/osssh.go
```
