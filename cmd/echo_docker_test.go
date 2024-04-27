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
const goVersion = "1.21"

var echoContainerName string

// prepareEchoContainer create a Docker Container for tcping2 echo server
func prepareEchoContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_ECHO_SERVER") != "" {
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

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")

	fmt.Printf("Try to build and start docker container  %s\n", echoContainerName)
	buildArgs := []docker.BuildArg{
		{
			Name:  "VENDOR_IMAGE_PREFIX",
			Value: vendorImagePrefix,
		},
		{
			Name:  "GO_VERSION",
			Value: goVersion,
		},
	}
	dockerContextDir := test.TestDir + "/../"
	if err != nil {
		err = fmt.Errorf("could not change to Docker Context Dir: %s", err)
		return
	}
	container, err = pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			BuildArgs:  buildArgs,
			ContextDir: dockerContextDir,
			Dockerfile: "test/Dockerfile",
		},
		&dockertest.RunOptions{
			Hostname:     echoContainerName,
			Name:         echoContainerName,
			ExposedPorts: []string{"8080/tcp"},
		}, func(config *docker.HostConfig) {
			// set AutoRemove to true so that stopped container goes away by itself
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})

	if err != nil {
		err = fmt.Errorf("error starting echo docker container: %v", err)
		return
	}
	// ip := container.Container.NetworkSettings.Networks[netlibNetworkName].IPAddress
	pool.MaxWait = echoContainerTimeout * time.Second
	host, port := common.GetContainerHostAndPort(container, "8080/tcp")
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
