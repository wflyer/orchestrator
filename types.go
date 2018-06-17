package main

type ContainerState struct {
	Name        string
	ContainerID string
	Node        string
	Status      string
	Addr        string
}

type ContainerStates []*ContainerState

func (c ContainerStates) Find(name string) *ContainerState {
	for _, cState := range c {
		if cState.Name == name {
			return cState
		}
	}
	return nil
}

type CreateContainerResponse struct {
	ContainerStates ContainerStates
	Status          string
	Reason          string
}

type NodeResponse struct {
	Name            string
	Addr            string
	ContainerStates ContainerStates
}

type NodeState struct {
	Name string
	Addr string
}

type NodeStates []*NodeState

func (n NodeStates) Find(name string) *NodeState {
	for _, nS := range n {
		if nS.Name == name {
			return nS
		}
	}
	return nil
}

type NodeUpdateRequest struct {
	Name                   string
	UpdatedContainerStates []*ContainerState
}

type CheckpointContainerRequest struct {
	Name   string
	ToNode string
}
type CheckpointContainerResponse struct {
	Name          string
	CheckpointID  string
	CheckpointDir string
}

type RestoreContainerRequest struct {
	Name          string
	FromNode      string
	CheckpointID  string
	CheckpointDir string
}
type RestoreContainerResponse struct {
	ContainerState ContainerState
}
