package main

import (
	"context"
	"fmt"

	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
)

// cmd: optional. ex. []string{"/app/server", "1"}
// exposedPorts: optional. ex. []string{"8080:8080/tcp"}
// returns containerID and err
func runContainer(name, image string, cmd, exposedPorts []string) (containerID string, err error) {
	cli, err := client.NewEnvClient()
	defer cli.Close()

	// expose ports
	portSet, portBindings, err := nat.ParsePortSpecs(exposedPorts)
	if err != nil {
		return
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		SecurityOpt:  []string{"seccomp:unconfined"},
	}

	// container configs
	config := &container.Config{
		Hostname:     "",
		Domainname:   "",
		User:         "",
		Image:        image,
		Cmd:          cmd,
		ExposedPorts: portSet,
	}

	fmt.Println("running container", name)

	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, name)
	if err != nil {
		return
	}
	containerID = resp.ID
	fmt.Println(containerID)

	startOpts := types.ContainerStartOptions{}
	err = cli.ContainerStart(context.Background(), containerID, startOpts)

	if err != nil {
		return
	}
	return
}

func getContainerIDByName(name string) string {
	cli, err := client.NewEnvClient()
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		for _, containerName := range container.Names {
			fmt.Println(containerName)
			if containerName == "/"+name {
				return container.ID
			}
		}
	}
	return ""
}

func removeContainer(name string) error {
	cli, err := client.NewEnvClient()
	defer cli.Close()

	containerID := getContainerIDByName(name)
	if containerID == "" {
		return errors.New("Cannot find container")
	}

	timeout := time.Second
	err = cli.ContainerStop(context.Background(), containerID, &timeout)
	if err != nil {
		return err
	}

	cli.ContainerRemove(context.Background(), containerID, types.ContainerRemoveOptions{})
	return nil
}

func checkpointContainer(name, checkpointID, checkpointDir string) error {
	containerID := getContainerIDByName(name)
	if containerID == "" {
		return errors.New("Cannot find container")
	}

	cli, err := client.NewEnvClient()
	defer cli.Close()
	if err != nil {
		return err
	}

	opts := types.CheckpointCreateOptions{
		CheckpointID:  checkpointID,
		CheckpointDir: checkpointDir,
		Exit:          true,
	}

	err = cli.CheckpointCreate(context.Background(), containerID, opts)
	if err != nil {
		return err
	}
	return nil
}

func restoreContainer(name, image string, cmd, exposedPorts []string, checkpointID, checkpointDir string) error {
	cli, err := client.NewEnvClient()
	defer cli.Close()
	if err != nil {
		return err
	}

	// expose ports
	portSet, portBindings, err := nat.ParsePortSpecs(exposedPorts)
	if err != nil {
		return err
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		SecurityOpt:  []string{"seccomp:unconfined"},
	}

	// container configs
	config := &container.Config{
		Hostname:     "",
		Domainname:   "",
		User:         "",
		Image:        "counter",
		Cmd:          cmd,
		ExposedPorts: portSet,
	}

	fmt.Println("running container", name)

	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, name)
	if err != nil {
		return err
	}
	containerID := resp.ID
	fmt.Println(containerID)

	opts := types.ContainerStartOptions{
		CheckpointID:  checkpointID,
		CheckpointDir: checkpointDir,
	}

	err = cli.ContainerStart(context.Background(), name, opts)
	if err != nil {
		return err
	}
	return nil
}
