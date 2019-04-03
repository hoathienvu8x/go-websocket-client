package gwsclient

import (
    "log"
    "sync"
    "errors"
    //"reflect"
    "net/url"
    "net/http"
    "crypto/tls"
    "github.com/gorilla/websocket"
)

type Socket struct {
    Conn              *websocket.Conn
    WebsocketDialer   *websocket.Dialer
    Url               string
    ConnectionOptions ConnectionOptions
    RequestHeader     http.Header
    OnConnected       func(socket Socket)
    OnTextMessage     func(message string, socket Socket)
    OnBinaryMessage   func(data [] byte, socket Socket)
    OnConnectError    func(err error, socket Socket)
    OnDisconnected    func(err error, socket Socket)
    OnPingReceived    func(data string, socket Socket)
    OnPongReceived    func(data string, socket Socket)
    IsConnected       bool
    sendMu            *sync.Mutex // Prevent "concurrent write to websocket connection"
    receiveMu         *sync.Mutex
}

type ConnectionOptions struct {
    UseCompression bool
    UseSSL         bool
    Proxy          func(*http.Request) (*url.URL, error)
    Subprotocols   [] string
}

// todo Yet to be done
type ReconnectionOptions struct {
}

func New(url string) Socket {
    return Socket{
        Url: url,
        RequestHeader: http.Header{},
        ConnectionOptions: ConnectionOptions{
            UseCompression: false,
            UseSSL:         true,
        },
        WebsocketDialer: &websocket.Dialer{},
        sendMu:          &sync.Mutex{},
        receiveMu:       &sync.Mutex{},
    }
}

func (socket *Socket) setConnectionOptions() {
    socket.WebsocketDialer.EnableCompression = socket.ConnectionOptions.UseCompression
    socket.WebsocketDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: socket.ConnectionOptions.UseSSL}
    socket.WebsocketDialer.Proxy = socket.ConnectionOptions.Proxy
    socket.WebsocketDialer.Subprotocols = socket.ConnectionOptions.Subprotocols
}

func (socket *Socket) Connect() {
    var err error;
    socket.setConnectionOptions()

    socket.Conn, _, err = socket.WebsocketDialer.Dial(socket.Url, socket.RequestHeader)

    if err != nil {
        log.Println("Error while connecting to server ", err)
        socket.IsConnected = false
        if socket.OnConnectError != nil {
            socket.OnConnectError(err, *socket)
        }
        return
    }
    if socket.OnConnected != nil {
        socket.IsConnected = true
        socket.OnConnected(*socket)
    }
    defaultPingHandler := socket.Conn.PingHandler()
    socket.Conn.SetPingHandler(func(appData string) error {
        if socket.OnPingReceived != nil {
            socket.OnPingReceived(appData, *socket)
        }
        return defaultPingHandler(appData)
    })
    defaultPongHandler := socket.Conn.PongHandler()
    socket.Conn.SetPongHandler(func(appData string) error {
        if socket.OnPongReceived != nil {
            socket.OnPongReceived(appData, *socket)
        }
        return defaultPongHandler(appData)
    })
    defaultCloseHandler := socket.Conn.CloseHandler()
    socket.Conn.SetCloseHandler(func(code int, text string) error {
        result := defaultCloseHandler(code, text)
        if socket.OnDisconnected != nil {
            socket.IsConnected = false
            socket.OnDisconnected(errors.New(text), *socket)
        }
        return result
    })
    go func() {
        for {
            socket.receiveMu.Lock()
            messageType, message, err := socket.Conn.ReadMessage()
            socket.receiveMu.Unlock()
            if err != nil {
                log.Println("read:", err)
                return
            }
            
            switch messageType {
            case websocket.TextMessage:
                if socket.OnTextMessage != nil {
                    socket.OnTextMessage(string(message), *socket)
                }
            case websocket.BinaryMessage:
                if socket.OnBinaryMessage != nil {
                    socket.OnBinaryMessage(message, *socket)
                }
            }
        }
    }()
}

func (socket *Socket) SendText(message string) {
    err := socket.send(websocket.TextMessage, [] byte (message))
    if err != nil {
        log.Println("write:", err)
        return
    }
}

func (socket *Socket) SendBinary(data [] byte) {
    err := socket.send(websocket.BinaryMessage, data)
    if err != nil {
        log.Println("write:", err)
        return
    }
}

func (socket *Socket) send(messageType int, data [] byte) error {
    socket.sendMu.Lock()
    err := socket.Conn.WriteMessage(messageType, data)
    socket.sendMu.Unlock()
    return err
}

func (socket *Socket) Close() {
    err := socket.send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
    if err != nil {
        log.Println("write close:", err)
    }
    socket.Conn.Close()
    if socket.OnDisconnected != nil {
        socket.IsConnected = false
        socket.OnDisconnected(err, *socket)
    }
}
