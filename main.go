package main

import (
	"bufio"
	"fmt"
	"github.com/ysugimoto/go-cliargs"
	"golang.org/x/net/websocket"
	"net/http"
	"os"
)

type AppHandler struct {
	connections map[string]*Connection
}

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

func (a *AppHandler) handleRemoteRequest(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Add("Content-Type", "text/plain")
	req.ParseForm()
	if msg := req.Form.Get("message"); msg == "" {
		a.Broadcast(msg)
		resp.WriteHeader(http.StatusOK)
		resp.Write([]byte("Message has accepted."))
	} else {
		resp.WriteHeader(http.StatusNotFound)
		resp.Write([]byte("Message is empty."))
	}
}

func (a *AppHandler) Broadcast(line string) {
	for _, c := range a.connections {
		if c.Type == READER {
			c.Send(line)
		}
	}
}

func main() {
	args := cliarg.NewArguments()
	args.Alias("", "stdin", nil)
	args.Alias("", "proxy", "")
	args.Alias("p", "port", "9000")
	args.Alias("", "proxy-server", nil)
	args.Alias("h", "help", nil)
	args.Alias("c", "client", "")
	args.Parse()

	if _, ok := args.GetOption("help"); ok {
		showUsage()
		os.Exit(0)
	}

	if c, _ := args.GetOptionAsString("client"); c != "" {
		if client, err := NewClient(c); err != nil {
			fmt.Println(err)
		} else {
			client.Listen()
		}
		return
	}

	var r *Remote
	if proxy, _ := args.GetOptionAsString("proxy"); proxy != "" {
		r = &Remote{URL: proxy}
	}

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

	if args.GetCommandSize() == 0 {
		showUsage()
		os.Exit(0)
	}

	app := &AppHandler{
		connections: make(map[string]*Connection),
	}
	if _, ok := args.GetOption("proxy-server"); !ok {
		file, _ := args.GetCommandAt(1)
		if r != nil {
			go startTail(file, r.Send)
		} else {
			go startTail(file, app.Broadcast)
		}
	}

	http.Handle("/", app)
	port, _ := args.GetOptionAsInt("port")
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		panic(err)
	}
}

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
