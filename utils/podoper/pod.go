package podoper

import (
	"context"
	"fmt"
	log "github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/watch"
	"k8s.io/kubernetes/pkg/client/conditions"
	"time"
)

type podoper struct {
	client coreclient.CoreV1Interface
}

func NewPodOper(client coreclient.CoreV1Interface) *podoper {
	return &podoper{client: client}
}

func (o *podoper) BuildPodWithParameters(kind, apiversion, podName, podNamespace, nodeName, image, probePath, volumeName, mountPath string, port int) *corev1.Pod {
	podName = fmt.Sprintf("%s-%s", podName, uuid.NewUUID())
	agentPod := &corev1.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       kind,
			APIVersion: apiversion,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      podName,
			Namespace: podNamespace,
		},
		Spec: corev1.PodSpec{
			Hostname:  podName,
			Subdomain: "test",
			NodeName:  nodeName,
			Containers: []corev1.Container{
				{
					Name:            podName,
					Image:           image,
					ImagePullPolicy: corev1.PullAlways,
					/*
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: probePath,
									Port: intstr.FromInt(port),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							SuccessThreshold:    1,
							TimeoutSeconds:      1,
							FailureThreshold:    3,
						},*/
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      volumeName,
							MountPath: mountPath,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							HostPort:      int32(port),
							ContainerPort: int32(port),
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: mountPath,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	return agentPod
}

func (o *podoper) LaunchPod(pod *corev1.Pod) (*corev1.Pod, error) {
	pod, err := o.client.Pods(pod.Namespace).Create(pod)
	if err != nil {
		return pod, err
	}
	watcher, err := o.client.Pods(pod.Namespace).Watch(v1.SingleObject(pod.ObjectMeta))
	if err != nil {
		return nil, err
	}
	// FIXME: hard code -> config
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	log.Infof("Waiting for pod %s to run...", pod.Name)
	event, err := watch.UntilWithoutRetry(ctx, watcher, conditions.PodRunning)
	if err != nil {
		log.Errorf("Error occurred while waiting for pod to run:%v", err)
		return nil, err
	}
	pod = event.Object.(*corev1.Pod)
	return pod, nil
}
