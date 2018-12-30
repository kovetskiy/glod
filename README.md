# Glöð

Glöð is dead-simple daemon which just stores key-value pair in memory, but with
one simple additional thing, it can remember PID of data owner. When PID dies,
key will be deleted.

## Starting server

```
glod server
```

Glöð will start listening for connections in unix socket `/var/run/user/$UID/glod.sock`.


## Set key

```
glod set mykey myvalue
```

or with pid
```
glod set mypidkey mypidvalue -p $BASHPID
```

## Get key

```
glod get mykey
```

or  
```
glod get mypidkey
```

then kill the pid and check it again.

### Install

```
go get github.com/kovetskiy/glod
```

### License
MIT
