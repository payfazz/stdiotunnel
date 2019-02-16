# stdiotunnel

Tunneling via `stdin` and `stdout`.

This project is useful for forwarding temporary port via `docker exec`.

Why not socat?, because socat cannot multiplex multiple connection with single stdio stream

## example

create fifo by

    mkfifo pipe1
    mkfifo pipe2

run server by

    ./stdiotunnel server tcp::8081 < pipe1 > pipe2

open another terminal, and run client by

    ./stdiotunnel client tcp:127.0.0.1:8091 > pipe1 < pipe2

every request to `:8081` will be forwarded to `127.0.0.1:8091`

## example forwarding ssh agent socket to container via `docker exec`

create fifo

    mkfifo p1 p2

run client

    ./stdiotunnel client unix:$SSH_AUTH_SOCK < p1 > p2

open another terminal, and run server

    docker exec -i <ctrid> sh -c 'rm -rf /tmp/ssh-agent; exec /stdiotunnel server unix:/tmp/ssh-agent' > p1 < p2

check if ssh agent works

    docker exec <ctrid> sh -c 'export SSH_AUTH_SOCK=/tmp/ssh-agent; exec ssh-add -L'

## KNOWN BUG

because this project using HTTP/2 protocol for multiplexing the connection, so it's limited to maximal stream available for single connection, see [this](https://httpwg.org/specs/rfc7540.html#StreamIdentifiers)
