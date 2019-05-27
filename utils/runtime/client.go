package runtime

import (
	"context"
	"fmt"
	log "github.com/Sirupsen/logrus"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	dockerapi "github.com/docker/docker/client"
	dockerstdcopy "github.com/docker/docker/pkg/stdcopy"
	"io"
	"io/ioutil"
	druntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubernetes/pkg/kubelet/dockershim/libdocker"
	"kubectl-test/utils/jsonmessage"
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

func (d *kubeDockerClient) getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d.timeout)
}

func (d *kubeDockerClient) PullImage(image string, stdout io.WriteCloser) error {
	ctx, cancel := d.getCancelableContext()
	defer cancel()
	resp, err := d.client.ImagePull(ctx, image, dockertypes.ImagePullOptions{})
	if nil != err {
		return err
	}
	defer resp.Close()
	jsonmessage.DisplayJSONMessagesStream(resp, stdout, 1, true, nil)
	return nil
}

func (d *kubeDockerClient) StartContainer(id string) error {
	ctx, cancel := d.getTimeoutContext()
	defer cancel()
	err := d.client.ContainerStart(ctx, id, dockertypes.ContainerStartOptions{})
	return err
}

func (d *kubeDockerClient) CreateContainer(image string, command []string, targetId string) (*dockercontainer.ContainerCreateCreatedBody, error) {
	ctx, cancel := d.getTimeoutContext()
	defer cancel()
	Config := &dockercontainer.Config{
		Entrypoint: strslice.StrSlice(command),
		Image:      image,
		OpenStdin:  true,
		Tty:        true,
		StdinOnce:  true,
	}
	HostConfig := &dockercontainer.HostConfig{
		NetworkMode: dockercontainer.NetworkMode(fmt.Sprintf("container:%s", targetId)),
		UsernsMode:  dockercontainer.UsernsMode(fmt.Sprintf("container:%s", targetId)),
		IpcMode:     dockercontainer.IpcMode(fmt.Sprintf("container:%s", targetId)),
		PidMode:     dockercontainer.PidMode(fmt.Sprintf("container:%s", targetId)),
	}
	createResp, err := d.client.ContainerCreate(ctx, Config, HostConfig, nil, "")
	return &createResp, err
}

func (d *kubeDockerClient) CleanContainer(id string) error {
	ctx, cancel := d.getTimeoutContext()
	defer cancel()
	statusCh, errCh := d.client.ContainerWait(ctx, id, dockercontainer.WaitConditionNotRunning)
	var rmErr error
	select {
	case err := <-errCh:
		if err != nil {
			log.Error("error waiting container exit, kill with --force")
			// timeout or error occurs, try force remove anywawy
			rmErr = d.RemoveContainer(id, true)
		}
	case <-statusCh:
		rmErr = d.RemoveContainer(id, false)
	}
	return rmErr
}

func (d *kubeDockerClient) RemoveContainer(id string, force bool) error {
	ctx, cancel := d.getTimeoutContext()
	defer cancel()
	opts := dockertypes.ContainerRemoveOptions{
		//RemoveVolumes: true,
		Force: force,
	}
	err := d.client.ContainerRemove(ctx, id, opts)
	return err
}

func (d *kubeDockerClient) AttachToContainer(containerId string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	HandleResizing(resize, func(size remotecommand.TerminalSize) {
		d.ResizeContainerTTY(containerId, uint(size.Height), uint(size.Width))
	})
	opts := dockertypes.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: false,
	}
	sopts := libdocker.StreamOptions{
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		RawTerminal:  tty,
	}
	ctx, cancel := d.getCancelableContext()
	defer cancel()
	resp, err := d.client.ContainerAttach(ctx, containerId, opts)
	if err != nil {
		return err
	}
	defer resp.Close()
	return d.holdHijackedConnection(sopts.RawTerminal, sopts.InputStream, sopts.OutputStream, sopts.ErrorStream, resp)
}

func (d *kubeDockerClient) holdHijackedConnection(tty bool, inputStream io.Reader, outputStream, errorStream io.Writer, resp dockertypes.HijackedResponse) error {
	receiveStdout := make(chan error)
	if outputStream != nil || errorStream != nil {
		go func() {
			receiveStdout <- d.redirectResponseToOutputStream(tty, outputStream, errorStream, resp.Reader)
		}()
	}

	stdinDone := make(chan struct{})
	go func() {
		if inputStream != nil {
			io.Copy(resp.Conn, inputStream)
		}
		resp.CloseWrite()
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		return err
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			return <-receiveStdout
		}
	}
	return nil
}

func (d *kubeDockerClient) ResizeContainerTTY(id string, height, width uint) error {
	ctx, cancel := d.getCancelableContext()
	defer cancel()
	return d.client.ContainerResize(ctx, id, dockertypes.ResizeOptions{
		Height: height,
		Width:  width,
	})
}

func (d *kubeDockerClient) redirectResponseToOutputStream(tty bool, outputStream, errorStream io.Writer, resp io.Reader) error {
	if outputStream == nil {
		outputStream = ioutil.Discard
	}
	if errorStream == nil {
		errorStream = ioutil.Discard
	}
	var err error
	if tty {
		_, err = io.Copy(outputStream, resp)
	} else {
		_, err = dockerstdcopy.StdCopy(outputStream, errorStream, resp)
	}
	return err
}

func HandleResizing(resize <-chan remotecommand.TerminalSize, resizeFunc func(size remotecommand.TerminalSize)) {
	if resize == nil {
		return
	}

	go func() {
		defer druntime.HandleCrash()

		for size := range resize {
			if size.Height < 1 || size.Width < 1 {
				continue
			}
			resizeFunc(size)
		}
	}()
}
