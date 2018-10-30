# stdiotunnel

Tunneling via `stdin` and `stdout`.

This project is useful for forwarding temporary port via `docker exec`.

## example

create fifo by

    mkfifo pipe1
    mkfifo pipe2

run server by

    ./stdiotunnel server tcp::8081 < pipe1 > pipe2

open another terminal, an run client by

    ./stdiotunnel client tcp:127.0.0.1:8091 > pipe1 < pipe2

every request to `:8081` will be forwarded to `127.0.0.1:8091`

## KNOWN BUG

because this project using HTTP/2 protocol for multiplexing the connection, so it's limited to maximal stream available for single connection, see [this](https://httpwg.org/specs/rfc7540.html#StreamIdentifiers)
