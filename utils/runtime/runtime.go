package runtime

import (
	"context"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
)

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
func (s *streamingRuntime) AttachContainer(name string, uid types.UID, containerId string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error {
	log.Infof("name:%v, uid:%v, containerId:%v, image:%v, commandSlice:%v", name, uid, containerId, s.image, s.commandSlice)
	stdout.Write([]byte(fmt.Sprintf("pulling image %s... \n\r", s.image)))
	err := s.client.PullImage(s.image, stdout)
	if nil != err {
		return err
	}
	stdout.Write([]byte("starting test container...\n\r"))
	res, err := s.client.CreateContainer(s.image, s.commandSlice, containerId)
	if nil != err {
		return err
	}
	err = s.client.StartContainer(res.ID)
	if nil != err {
		return err
	}
	defer s.client.CleanContainer(containerId)
	stdout.Write([]byte("container created, open tty...\n\r"))
	err = s.client.AttachToContainer(containerId, stdin, stdout, stderr, tty, resize)
	if nil != err {
		return err
	}
	return nil
}
