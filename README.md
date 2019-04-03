# Simple webSocket client with golang

Simple websocket client with golang

## The sample code

```go
package main

import (
    "os"    
    "log"
    "os/signal"
    "github.com/hoathienvu8x/gwsclient"
)

func main() {
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    socket := gwsclient.New("wss://hoathienvu8x.local/realtime");
    socket.OnConnected = func(socket gwsclient.Socket) {
        log.Println("Connected to server");
    };
    
    socket.OnConnectError = func(err error, socket gwsclient.Socket) {
        log.Println("Recieved connect error ", err)
    };
    
    socket.OnTextMessage = func(message string, socket gwsclient.Socket) {
        log.Println("Recieved message " + message)
    };
    
    socket.OnBinaryMessage = func(data [] byte, socket gwsclient.Socket) {
        log.Println("Recieved binary data ", data)
    };
    
    /*socket.OnPingReceived = func(data string, socket gwsclient.Socket) {
        log.Println("Recieved ping " + data)
    };*/
    
    socket.OnPongReceived = func(data string, socket gwsclient.Socket) {
        log.Println("Recieved pong " + data)
    };
    
    socket.OnDisconnected = func(err error, socket gwsclient.Socket) {
        log.Println("Disconnected from server ")
        return
    };
    
    socket.Connect()
    
    for {
        select {
        case <-interrupt:
            log.Println("interrupt")
            socket.Close()
            return
        }
    }
}

```