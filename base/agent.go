package base

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"kubectl-test/config"
	//"kubectl-test/config/routes"
	"context"
	remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	remotecommandserver "k8s.io/kubernetes/pkg/kubelet/server/remotecommand"
	"kubectl-test/utils/runtime"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

const dockerContainerPrefix = "docker://"

type TestServer struct {
	config *config.Options
}

func NewTestServer(config *config.Options) *TestServer {
	return &TestServer{config: config}
}

func (t *TestServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/api/test", t.serveTest)
	mux.HandleFunc("/v1/api/health", t.health)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", t.config.Port), mux); nil != err {
		log.Fatal(err.Error())
		os.Exit(5)
	}
}

func (t *TestServer) serveTest(w http.ResponseWriter, req *http.Request) {
	log.Infof("req:%v", req)
	containerId := req.FormValue("containerid")
	if len(containerId) < 1 {
		http.Error(w, "target container id must be provided", 400)
		return
	}
	if !strings.HasPrefix(containerId, dockerContainerPrefix) {
		http.Error(w, "only docker container is suppored right now", 400)
	}
	dockerContainerId := containerId[len(dockerContainerPrefix):]

	image := req.FormValue("image")
	if len(image) < 1 {
		http.Error(w, "image must be provided", 400)
		return
	}
	command := req.FormValue("command")
	var commandSlice []string
	err := json.Unmarshal([]byte(command), &commandSlice)
	if err != nil || len(commandSlice) < 1 {
		http.Error(w, "cannot parse command", 400)
		return
	}
	log.Infof("%v, %v, %v", dockerContainerId, image, commandSlice)
	streamOpts := &remotecommandserver.Options{
		Stdin:  true,
		Stdout: true,
		Stderr: false,
		TTY:    true,
	}
	cxt, cancel := context.WithCancel(req.Context())
	defer cancel()
	streamingRuntime, err := runtime.NewStreamRuntime(image, commandSlice, cxt, cancel)
	if nil != err {
		http.Error(w, fmt.Sprintf("streaming runtime err:%v", err), 400)
		return
	}
	remotecommandserver.ServeAttach(
		w,
		req,
		streamingRuntime, //runtime,
		"",               // unused: podName
		"",               // unusued: podUID
		dockerContainerId,
		streamOpts,
		t.config.StreamIdleTimeout,
		t.config.StreamCreationTimeout,
		remotecommandconsts.SupportedStreamingProtocols)
}

func (t *TestServer) health(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("I'm OK!"))
}

func Agent(c *cli.Context) {
	var (
		err     error
		options *config.Options
	)
	options, err = config.ParseConf(c)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	ts := NewTestServer(options)
	go ts.Start()
	waitingForExit()
}

func waitingForExit() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	killing := false
	for range sc {
		if killing {
			log.Info("Second interrupt: exiting")
			os.Exit(1)
		}
		killing = true
		go func() {
			log.Info("Interrupt: closing down...")
			log.Info("done")
			os.Exit(1)
		}()
	}
}
