package cmd

import (
	"fmt"
	"net"
	"os"
	"tcping2/test"
	"time"

	"github.com/tommi2day/gomodules/common"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const echoContainerTimeout = 10

var vendorImagePrefix = os.Getenv("VENDOR_IMAGE_PREFIX")
var goImage = common.GetEnv("GO_IMAGE", vendorImagePrefix+"docker.io/library/golang:1.22.2-bookworm")
var runtimeImage = common.GetEnv("RUNTIME_IMAGE", vendorImagePrefix+"docker.io/library/debian:bookworm")
var echoContainerName string

// prepareEchoContainer create a Docker Container for tcping2 echo server
func prepareEchoContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_ECHO_CONTAINER") != "" {
		err = fmt.Errorf("skipping Echo Server Container in CI environment")
		return
	}
	echoContainerName = os.Getenv("ECHO_CONTAINER_NAME")
	if echoContainerName == "" {
		echoContainerName = "tcping2-echo-server"
	}
	var pool *dockertest.Pool
	pool, err = common.GetDockerPool()
	if err != nil {
		return
	}

	fmt.Printf("Try to build and start docker container  %s\n", echoContainerName)
	buildArgs := []docker.BuildArg{
		{
			Name:  "RUNTIME_IMAGE",
			Value: runtimeImage,
		},
		{
			Name:  "GO_IMAGE",
			Value: goImage,
		},
	}
	dockerContextDir := test.TestDir + "/../"
	container, err = pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "docker/image/Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     echoContainerName,
			Name:         echoContainerName,
			ExposedPorts: []string{"8080/tcp"},
			Entrypoint:   []string{"/tcping2", "echo", "--server", "--port", "8080", "--timeout", "15"},
		}, func(config *docker.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})

	if err != nil {
		err = fmt.Errorf("error starting echo docker container: %v", err)
		return
	}
	// wait to proceed
	time.Sleep(10 * time.Second)
	pool.MaxWait = echoContainerTimeout * time.Second
	host, port := common.GetContainerHostAndPort(container, "8080/tcp")
	if host == "" || port == 0 {
		err = fmt.Errorf("could not get container host and port")
		return
	}
	fmt.Printf("Wait to successfully connect to Echo Server to %s:%d (max %ds)...\n", host, port, echoContainerTimeout)
	start := time.Now()
	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			fmt.Printf("Err:%s\n", err)
		}
		return err
	}); err != nil {
		fmt.Printf("Could not connect to Echo Server Container: %d", err)
		return
	}
	_ = c.Close()
	echoHost = host
	echoPort = fmt.Sprintf("%d", port)

	// wait 5s to init container
	time.Sleep(5 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("Echo Container is available after %s\n", elapsed.Round(time.Millisecond))
	return
}

func destroyEchoContainer(container *dockertest.Resource) {
	common.DestroyDockerContainer(container)
}
