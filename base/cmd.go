package base

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	dockerterm "github.com/docker/docker/pkg/term"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"kubectl-test/config"
	"kubectl-test/utils/term"
	"net/url"
	"os"
)

func SetupTTY() term.TTY {
	t := term.TTY{}
	t.Raw = true
	stdin, stdout, _ := dockerterm.StdStreams()
	t.In = stdin
	t.Out = stdout
	return t
}

func getContainerIdByName(pod *corev1.Pod, containerName string) (string, error) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name != containerName {
			continue
		}
		if !containerStatus.Ready {
			return "", fmt.Errorf("container %s id not ready", containerName)
		}
		return containerStatus.ContainerID, nil
	}
	return "", fmt.Errorf("cannot find specified container %s", containerName)
}

type DefaultRemoteExecutor struct{}

func (*DefaultRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	})
}

func Cmd(c *cli.Context) {
	var (
		err     error
		options *config.Options
	)
	options, err = config.ParseConf(c)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	} else {
		log.Infof("options:%v", options)
	}
	//streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	configFlag := &genericclioptions.ConfigFlags{}
	clientConfig, _ := configFlag.ToRESTConfig()
	clientset, err := kubernetes.NewForConfig(clientConfig)
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	pod, err := clientset.CoreV1().Pods(options.Namespace).Get(options.PodName, metav1.GetOptions{})
	if nil != err {
		log.Fatal(err.Error())
		os.Exit(1)
	}
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		log.Fatal(fmt.Sprintf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase))
		os.Exit(1)
	}

	hostIP := pod.Status.HostIP
	if len(pod.Spec.Containers) > 1 {
		usageString := fmt.Sprintf("Defaulting container name to %s.", pod.Spec.Containers[0].Name)
		log.Infof("%s", usageString)
	}
	containerName := pod.Spec.Containers[0].Name
	containerId, err := getContainerIdByName(pod, containerName)
	if "" == containerId {
		log.Fatal(fmt.Sprintf("%v, %v, containerId is nil.", pod, containerName))
		os.Exit(1)
	}
	t := SetupTTY()
	var sizeQueue remotecommand.TerminalSizeQueue
	if t.Raw {
		sizeQueue = t.MonitorSize(t.GetSize())
	}
	var ErrOut io.Writer = nil
	fn := func() error {
		uri, err := url.Parse(fmt.Sprintf("http://%s:%d", hostIP, options.Port))
		if nil != err {
			return err
		}
		//uri.Path = fmt.Sprintf("/v1/api/test")
		uri.Path = fmt.Sprintf("/api/v1/debug")
		params := url.Values{}
		params.Add("image", options.Image)
		params.Add("containerid", containerId)
		params.Add("container", containerId)
		bytes, _ := json.Marshal([]string{options.Command})
		params.Add("command", string(bytes))
		uri.RawQuery = params.Encode()
		log.Infof("image:%v, containerid:%v, command:%v", options.Image, containerId, options.Command)
		return (&DefaultRemoteExecutor{}).Execute("POST", uri, clientConfig, t.In, t.Out, ErrOut, t.Raw, sizeQueue)
	}

	if err := t.Safe(fn); err != nil {
		log.Fatalf("%v", err)
		os.Exit(1)
	}
}
