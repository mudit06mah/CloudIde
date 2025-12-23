package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type TerminalMessage struct{
	Op string `json:"op"`
	Data string `json:"data"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

type TerminalSession struct{
	ws *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}

	readBuf []byte
	readMu	sync.Mutex
}

func (t *TerminalSession) Next() *remotecommand.TerminalSize{
	select{
	case size := <-t.sizeChan:
		return &size
	case <- t.doneChan:
		return nil
	}
}

func (t *TerminalSession) Read(p []byte) (int, error){
	_,message,err := t.ws.ReadMessage()
	if err != nil{
		return 0,err
	}

	var msg TerminalMessage
	if err = json.Unmarshal(message, &msg); err != nil{
		return copy(p,message), nil
	}

	switch msg.Op{
	case "stdin":
		return copy(p,[]byte(msg.Data)),nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
		return 0, nil
	default:
		return 0, nil
	}
}

func (t *TerminalSession) Write(p []byte) (int, error){
	err := t.ws.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

func HandleTerminal(w http.ResponseWriter, r *http.Request, client *kubernetes.Clientset, config *rest.Config, podname string){
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {return true},
	}
	conn,err := upgrader.Upgrade(w, r, nil)
	if err != nil{
		return
	}
	defer conn.Close()

	session := &TerminalSession{
		ws: conn,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
	}


	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podname).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: "shell",
			Command: []string{"/bin/bash"},
			Stdin: true,
			Stdout: true,
			Stderr: true,
			TTY: true,
		}, scheme.ParameterCodec)

	exec,err := remotecommand.NewSPDYExecutor(config,"POST",req.URL())
	if err != nil{
		fmt.Println("Error creating SPDY executor:", err)
		return
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin: session,
		Stdout: session,
		Stderr: session,
		TerminalSizeQueue: session,
		Tty: true,
	})

	if err != nil{
		fmt.Println("Stream error:", err)
	}

}