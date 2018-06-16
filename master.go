package main

import (
	"net/http"

	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type ContainerState struct {
	Name        string
	ContainerID string
	Node        string
	Status      string
}

var containerStates []*ContainerState

var containerStatesLock = sync.RWMutex{}

func listContainerRequest(c echo.Context) error {
	containerStatesLock.RLock()
	defer containerStatesLock.RUnlock()
	return c.JSON(http.StatusOK, containerStates)
}

type CreateContainerResponse struct {
	ContainerStates []*ContainerState
	Status          string
	Reason          string
}

func createContainerRequest(c echo.Context) error {
	name := c.Param("name")

	containerStatesLock.Lock()
	defer containerStatesLock.Unlock()
	containerState := &ContainerState{
		Name:        name,
		ContainerID: "",
		Node:        "",
		Status:      "pending",
	}
	containerStates = append(containerStates, containerState)

	resp := &CreateContainerResponse{
		ContainerStates: containerStates,
		Status:          "ok",
		Reason:          "accepted",
	}
	return c.JSON(http.StatusAccepted, resp)
}

func deleteContainerRequest(c echo.Context) error {
	name := c.Param("name")

	containerStatesLock.Lock()
	defer containerStatesLock.Unlock()
	for _, state := range containerStates {
		if state.Name == name {
			state.Status = "deleting"
		}
	}
	resp := &CreateContainerResponse{
		ContainerStates: containerStates,
		Status:          "ok",
		Reason:          "accepted",
	}
	return c.JSON(http.StatusAccepted, resp)
}

type NodeState struct {
	Name string
	Addr string
}

var nodeStates []*NodeState

var nodeStatesLock = sync.RWMutex{}

func registerNodeRequest(c echo.Context) error {
	name := c.Param("name")
	addr := c.FormValue("addr")
	nodeStatesLock.Lock()
	defer nodeStatesLock.Unlock()

	exists := false
	for _, state := range nodeStates {
		if state.Name == name {
			exists = true

		}
	}
	if !exists {
		nodeState := &NodeState{
			Name: name,
			Addr: addr,
		}
		nodeStates = append(nodeStates, nodeState)
	}

	return c.JSON(http.StatusOK, nodeStates)
}

func main() {
	containerStates = make([]*ContainerState, 0)
	nodeStates = make([]*NodeState, 0)
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/containers", listContainerRequest)
	e.POST("/containers/:name", createContainerRequest)
	e.DELETE("/containers/:name", deleteContainerRequest)
	e.POST("/node/:name", registerNodeRequest)
	e.Logger.Fatal(e.Start(":1323"))
}
