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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"os/signal"
	//"k8s.io/kubernetes/pkg/util/interrupt"
	"kubectl-test/config"
	"kubectl-test/utils/podoper"
	"kubectl-test/utils/term"
	"net/url"
	"os"
)

func LaunchAgentPod(client coreclient.CoreV1Interface, nodename string, podNamespace string, port int) (*corev1.Pod, error) {
	var agentPod *corev1.Pod
	var err error
	op := podoper.NewPodOper(client)
	agentPodkind := "Pod"
	agentApiVersion := "v1"
	agentPodName := "test"
	agentPodNamespace := podNamespace
	agentNodeName := nodename
	agentImage := "fifa2002nb/kubectltest:latest"
	agentProbePath := "/health"
	agentVolumeName := "docker"
	agentMountName := "/var/run/docker.sock"
	agentPort := port
	agentPod = op.BuildPodWithParameters(agentPodkind, agentApiVersion, agentPodName, agentPodNamespace, agentNodeName, agentImage, agentProbePath, agentVolumeName, agentMountName, agentPort)
	agentPod, err = op.LaunchPod(agentPod)
	if err != nil {
		return nil, err
	}
	return agentPod, nil
}

func BuildAgentUri(hostIP string, port int, image, containerid, command string) (*url.URL, error) {
	uri, err := url.Parse(fmt.Sprintf("http://%s:%d", hostIP, port))
	if nil != err {
		return nil, err
	}
	uri.Path = fmt.Sprintf("/v1/api/test")
	params := url.Values{}
	params.Add("image", image)
	params.Add("containerid", containerid)
	bytes, _ := json.Marshal([]string{command})
	params.Add("command", string(bytes))
	uri.RawQuery = params.Encode()
	return uri, err
}

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

	var agentPod *corev1.Pod = nil
	if options.Agentless {
		agentPod, err := LaunchAgentPod(clientset.CoreV1(), pod.Spec.NodeName, options.Namespace, options.Port)
		if nil != err {
			log.Fatalf("%v, %v", agentPod, err)
			os.Exit(1)
		}
	}

	t := SetupTTY()
	var sizeQueue remotecommand.TerminalSizeQueue
	if t.Raw {
		sizeQueue = t.MonitorSize(t.GetSize())
	}
	var ErrOut io.Writer = nil

	fn := func() error {
		uri, err := BuildAgentUri(hostIP, options.Port, options.Image, containerId, options.Command)
		if nil != err {
			return err
		}
		log.Infof("image:%v, containerid:%v, command:%v", options.Image, containerId, options.Command)
		return (&DefaultRemoteExecutor{}).Execute("POST", uri, clientConfig, t.In, t.Out, ErrOut, t.Raw, sizeQueue)
	}
	/*
		fnWithCleanUp := func() error {
			return interrupt.Chain(nil, func() {
				if options.Agentless && nil != agentPod {
					log.Infof("Start deleting agent pod %s", agentPod.Name)
					err := clientset.CoreV1().Pods(agentPod.Namespace).Delete(agentPod.Name, v1.NewDeleteOptions(0))
					if nil != err {
						log.Errorf("failed to delete agent pod[Name:%s, Namespace: %s], consider manual deletion.", agentPod.Name, agentPod.Namespace)
					}
				}
			}).Run(fn)
		}
	*/
	if err := t.Safe(fn); err != nil {
		log.Fatalf("%v", err)
	}

	cleanUp := func() {
		if options.Agentless && nil != agentPod {
			log.Infof("Start deleting agent pod %s", agentPod.Name)
			err := clientset.CoreV1().Pods(agentPod.Namespace).Delete(agentPod.Name, v1.NewDeleteOptions(0))
			if nil != err {
				log.Errorf("failed to delete agent pod[Name:%s, Namespace: %s], consider manual deletion.", agentPod.Name, agentPod.Namespace)
			}
		}
	}
	waitingForExit(cleanup)
}

func waitingForExit(fn func()) {
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
			fn()
			log.Info("done")
			os.Exit(1)
		}()
	}
}
