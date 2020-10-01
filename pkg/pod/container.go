package pod

import (
	"bufio"
	"bytes"
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func filterNotReadyContainers(css []corev1.ContainerStatus) []corev1.ContainerStatus {
	var notReadyContainers []corev1.ContainerStatus
	for _, cs := range css {
		if !cs.Ready {
			notReadyContainers = append(notReadyContainers, cs)
		}
	}
	return notReadyContainers
}

func getContainerLog(c *kubernetes.Clientset, ns, pname, cname string) (string, error) {
	var tailN = int64(15)
	req := c.CoreV1().Pods(ns).GetLogs(pname, &corev1.PodLogOptions{
		TailLines: &tailN,
		Container: cname,
	})

	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer readCloser.Close()

	var (
		line int
		buf  bytes.Buffer
		r    = bufio.NewScanner(readCloser)
	)
	for r.Scan() {
		buf.Write(r.Bytes())
		buf.WriteString("\n")
		line++
	}
	if line == 0 {
		buf.WriteString("<none>\n")
	}
	return buf.String(), nil
}
