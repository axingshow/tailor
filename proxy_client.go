package main

import (
	"fmt"
	"golang.org/x/net/websocket"
)

func createProxyClient(app *AppHandler) websocket.Handler {
	return func(conn *websocket.Conn) {
		fmt.Println("Proxy Connection")
		c := NewConnection(PROXY, conn)
		c.OnClose = func() {
			delete(app.connections, c.Id)
		}
		c.OnMessage = func(p Payload) {
			app.Broadcast(p)
		}

		app.connections[c.Id] = c
		c.Poll()
	}
}
