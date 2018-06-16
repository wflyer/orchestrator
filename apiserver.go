package main

import (
	"net/http"
	"sync"

	"fmt"

	"encoding/json"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
)

func listContainerRequest(c echo.Context) error {
	containerStatesLock.RLock()
	defer containerStatesLock.RUnlock()
	return c.JSON(http.StatusOK, containerStates)
}

func createContainerRequest(c echo.Context) error {
	name := c.Param("name")

	containerStatesLock.Lock()
	defer containerStatesLock.Unlock()

	node, err := FindAllocNode("")
	var nodename string
	if err != nil {
		nodename = ""
	} else {
		nodename = node.Name
	}
	containerState := &ContainerState{
		Name:        name,
		ContainerID: "",
		Node:        nodename,
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

// assumes lock already acquired
func generateNodeResponse(node *NodeState) NodeResponse {
	return NodeResponse{
		Name:            node.Name,
		Addr:            node.Addr,
		ContainerStates: node.ContainerStates(),
	}
}

func getNodeRequest(c echo.Context) error {
	name := c.Param("name")
	nodeStatesLock.Lock()
	defer nodeStatesLock.Unlock()

	var nodeState *NodeState
	for _, state := range nodeStates {
		if state.Name == name {
			nodeState = state
		}
	}
	if nodeState == nil {
		return c.String(http.StatusNotFound, "Not found")
	}

	resp := generateNodeResponse(nodeState)
	return c.JSON(http.StatusOK, resp)
}

func registerNodeRequest(c echo.Context) error {
	name := c.Param("name")
	addr := c.FormValue("addr")
	nodeStatesLock.Lock()
	defer nodeStatesLock.Unlock()

	var nodeState *NodeState
	for _, state := range nodeStates {
		if state.Name == name {
			nodeState = state
		}
	}
	if nodeState == nil {
		nodeState = &NodeState{
			Name: name,
			Addr: addr,
		}
		nodeStates = append(nodeStates, nodeState)
	}

	resp := generateNodeResponse(nodeState)
	fmt.Println(resp)

	return c.JSON(http.StatusOK, resp)
}

func updateNodeRequest(c echo.Context) error {
	req := new(NodeUpdateRequest)
	if err := c.Bind(req); err != nil {
		return err
	}

	containerStatesLock.Lock()
	defer containerStatesLock.Unlock()

	// find container and update state.
	for _, newCState := range req.UpdatedContainerStates {
		for _, oldCState := range containerStates {
			if newCState.Name == oldCState.Name {
				fmt.Println("update container: old: ", oldCState, "new: ", newCState)
				if newCState.Status == "running" {
					if oldCState.Status != "running" && newCState.Addr != "" {
						log.Info("add new container to lb: ", newCState.Addr)
						proxyTarget.LB.Add(newCState.Addr)
					}
				} else {
					log.Info("remove container from lb: ", oldCState.Addr)
					proxyTarget.LB.Remove(oldCState.Addr)
				}

				oldCState.Status = newCState.Status
				oldCState.ContainerID = newCState.ContainerID
				oldCState.Node = newCState.Node
				oldCState.Addr = newCState.Addr

			}
			break
		}
	}

	b, _ := json.Marshal(containerStates)
	log.Debug("updated container states:", string(b))

	return c.String(http.StatusOK, "{}")
}

func apiServer(wg sync.WaitGroup) {
	defer wg.Done()
	e := echo.New()
	// Middleware
	e.Use(middleware.Logger())
	//e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.GET("/containers", listContainerRequest)
	e.POST("/containers/:name", createContainerRequest)
	e.DELETE("/containers/:name", deleteContainerRequest)

	e.GET("/node/:name", getNodeRequest)
	e.POST("/node/:name", registerNodeRequest)
	e.POST("/node/:name/update", updateNodeRequest)

	e.Logger.Fatal(e.Start(":1323"))
}
