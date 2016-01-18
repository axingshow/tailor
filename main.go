// Tailor: Anywhere log casting
//
// @author Yoshiaki Sugimoto
// @license MIT
package main

import (
	"bufio"
	"fmt"
	"github.com/ysugimoto/go-cliargs"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
)

// Application handler
// Handle HTTP request, upgrading WebSocket request,
// with managing connections if working server.
type AppHandler struct {
	// WebSocket connection instances
	// key: string connection id
	// Connection *Connection conenction instance
	connections map[string]*Connection
}

// Implements http.Handler interface
// Serving HTTP request, or upgrading to WebSocket by segment.
func (a *AppHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var handler websocket.Handler

	switch req.URL.Path {
	case "/remote":
		a.handleRemoteRequest(resp, req)
		return
	//case "/proxy":
	//	handler = createProxyClient(a)
	default:
		handler = createReader(a)
	}
	ws := websocket.Server{Handler: handler}
	ws.ServeHTTP(resp, req)
}

// Accept remote messaging
// Need to POST request, and message field.
func (a *AppHandler) handleRemoteRequest(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Add("Content-Type", "text/plain")
	req.ParseForm()
	if msg := req.Form.Get("message"); msg == "" {
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("Message is empty."))
	} else {
		fmt.Println("Message incoming", msg)
		a.Broadcast(msg)
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte("Message has accepted."))
	}
}

// Boardcast all websocket connections
// cast message only READER type connection.
func (a *AppHandler) Broadcast(line string) {
	for _, c := range a.connections {
		if c.Type == READER {
			c.Send(line)
		}
	}
}

// main function
// parse command-line arguments,
// and switch working mode
func main() {
	args := cliarg.NewArguments()
	args.Alias("s", "stdin", nil)
	args.Alias("P", "proxy", "")
	args.Alias("p", "port", "9000")
	args.Alias("", "proxy-server", nil)
	args.Alias("h", "help", nil)
	args.Alias("c", "client", "")
	args.Parse()

	// if help flag supplied, show usage
	if _, ok := args.GetOption("help"); ok {
		showUsage()
		os.Exit(0)
	}

	// working client mode
	if c, _ := args.GetOptionAsString("client"); c != "" {
		if client, err := NewClient(c); err != nil {
			fmt.Println(err)
		} else {
			client.Listen()
		}
		return
	}

	// Create remote object if proxy option supplied
	var r *Remote
	if proxy, _ := args.GetOptionAsString("proxy"); proxy != "" {
		r = &Remote{URL: proxy}
	}

	// read and cast from stdin
	if _, ok := args.GetOption("stdin"); ok {
		fmt.Println("Read from stdin")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if r != nil {
				r.Send(scanner.Text())
			} else {
				os.Stdout.WriteString(scanner.Text() + "\n")
			}
		}
		return
	}

	app := &AppHandler{
		connections: make(map[string]*Connection),
	}

	// Run with proxy-server mode
	if _, ok := args.GetOption("proxy-server"); ok {
		http.Handle("/", app)
		port, _ := args.GetOptionAsInt("port")
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			panic(err)
		}
		return
	}

	// if command argument is nothing, show usage
	if args.GetCommandSize() == 0 {
		showUsage()
		os.Exit(0)
	}

	// Tailing file
	file, _ := args.GetCommandAt(1)
	if r != nil {
		startTail(file, r.Send)
	} else {
		go startTail(file, app.Broadcast)
		// serving HTTP
		http.Handle("/", app)
		port, _ := args.GetOptionAsInt("port")
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			panic(err)
		}
	}

}

// Show usage
func showUsage() {
	help := `========================================
tailor: the realtime logging transporter
========================================
Usage:
  $ tailor [options] file

Options
  -p, --port        : Listen port number if works server
  -h, --help        : Show this help
      --stdin       : Get data from stdin
      --proxy       : Send data to proxy server
      --proxy-server: Work with proxy-server`
	fmt.Println(help)
}
