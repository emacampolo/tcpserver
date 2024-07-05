# tcpserver

By default, the server listens on port 8080 and is configured to echo back any message it receives.
After handling a message, the server will close the connection.

```
go run example/main.go
```

```
telnet 127.0.0.1 8080
```