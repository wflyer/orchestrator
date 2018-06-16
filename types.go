package main

type ContainerState struct {
	Name        string
	ContainerID string
	Node        string
	Status      string
}

type CreateContainerResponse struct {
	ContainerStates []*ContainerState
	Status          string
	Reason          string
}

type NodeResponse struct {
	Name            string
	Addr            string
	ContainerStates []*ContainerState
}

type NodeState struct {
	Name string
	Addr string
}

type NodeStates []*NodeState

type NodeUpdateRequest struct {
	Name                   string
	UpdatedContainerStates []*ContainerState
}
