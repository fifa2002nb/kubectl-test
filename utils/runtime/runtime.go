package runtime

import (
	"context"
	dockerapi "github.com/docker/docker/client"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
	"time"
)

const (
	DockerTimeout  = 30 * time.Second
	DockerEndpoint = "unix:///var/run/docker.sock"
)

type kubeDockerClient struct {
	timeout time.Duration
	client  *dockerapi.Client
}

func NewKubeDockerClient() (*kubeDockerClient, error) {
	var err error
	kdc := &kubeDockerClient{timeout: DockerTimeout}
	kdc.client, err = dockerapi.NewClient(DockerEndpoint, "", nil, nil)
	if nil != err {
		return nil, err
	}
	return kdc, nil
}

func (d *kubeDockerClient) getCancelableContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func (d *kubeDockerClient) PullImage(image string) {
}

func (d *kubeDockerClient) StartContainer()    {}
func (d *kubeDockerClient) CreateContainer()   {}
func (d *kubeDockerClient) CleanContainer()    {}
func (d *kubeDockerClient) RmContainer()       {}
func (d *kubeDockerClient) AttachToContainer() {}

type streamingRuntime struct {
	client       *kubeDockerClient
	image        string
	commandSlice []string
	cxt          context.Context
	cancel       context.CancelFunc
}

func NewStreamRuntime(image string, commandSlice []string, cxt context.Context, cancel context.CancelFunc) (*streamingRuntime, error) {
	client, err := NewKubeDockerClient()
	s := &streamingRuntime{client: client, image: image, commandSlice: commandSlice, cxt: cxt, cancel: cancel}
	return s, err
}
func (s *streamingRuntime) AttachContainer(name string, uid types.UID, container string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	return nil
}
