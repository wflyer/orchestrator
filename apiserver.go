package main

import (
	"net/http"
	"sync"

	"fmt"

	"encoding/json"

	"bytes"

	"io/ioutil"

	"time"

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

func migrateContainerRequest(c echo.Context) error {
	name := c.Param("name")

	fmt.Println("========== received request", time.Now().UnixNano())

	containerStatesLock.Lock()
	defer containerStatesLock.Unlock()

	// get container info
	var containerState *ContainerState
	for _, cState := range containerStates {
		if cState.Name == name {
			containerState = cState
		}
	}

	if containerState == nil {
		return c.String(http.StatusNotFound, "Not found")
	}

	// find alternative node except current node
	newNode, err := FindAllocNode(containerState.Node)
	if err != nil {
		log.Error("Cannot find affordable node for container", containerState.Name, "current: ", containerState.Node)
		return c.String(http.StatusRequestedRangeNotSatisfiable, "Alternative node not available")
	}

	// request checkpoint to original worker node
	checkpointReq := CheckpointContainerRequest{
		Name:   name,
		ToNode: newNode.Name,
	}
	bodyB, err := json.Marshal(checkpointReq)
	log.Debug("request checkpoint to Node:", string(bodyB))
	if err != nil {
		panic(err)
	}

	// find node addr
	oldNodeAddr := nodeStates.Find(containerState.Node).Addr

	var wg sync.WaitGroup
	wg.Add(2)

	var restoreReq RestoreContainerRequest
	go func() {
		defer wg.Done()
		body := bytes.NewReader(bodyB)
		apiAddr := oldNodeAddr + "/checkpoint/" + containerState.Name
		fmt.Println(">>>> request checkpoint to", apiAddr)

		resp, err := http.Post(
			apiAddr, "application/json", body)

		if err != nil {
			panic(fmt.Sprintf("checkpoint req error: ", err))
		}
		if resp.StatusCode != http.StatusOK {
			panic(fmt.Sprintf("checkpoint req error: not expected status ", resp.StatusCode))
		}

		defer resp.Body.Close()
		fmt.Println("========== received checkpoint finished", time.Now().UnixNano())

		bodyResp, err := ioutil.ReadAll(resp.Body)
		log.Debug("checkpoint resp from worker", string(bodyResp))

		// remove old addr
		proxyTarget.LB.Remove(containerState.Addr)

		checkpointResp := new(CheckpointContainerResponse)
		json.Unmarshal(bodyResp, &checkpointResp)

		// request restore to new worker node
		restoreReq = RestoreContainerRequest{
			Name:          name,
			FromNode:      newNode.Name,
			CheckpointID:  checkpointResp.CheckpointID,
			CheckpointDir: checkpointResp.CheckpointDir,
		}
	}()

	go func() {
		defer wg.Done()
		restoreCreateReq := RestoreContainerRequest{
			Name:     name,
			FromNode: newNode.Name,
		}

		restoreReqBodyB, err := json.Marshal(restoreCreateReq)
		log.Debug("request restore to Node:", string(restoreReqBodyB))
		if err != nil {
			panic(err)
		}
		body := bytes.NewReader(restoreReqBodyB)
		apiAddr := newNode.Addr + "/restorecreate/" + containerState.Name

		fmt.Println(">>>> request restore to", apiAddr)

		resp, err := http.Post(
			apiAddr, "application/json", body)

		if err != nil {
			fmt.Println("restore req error: ", err)
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Println("restore req error: not expected status ", resp.StatusCode)
		}
		defer resp.Body.Close()

		bodyResp, err := ioutil.ReadAll(resp.Body)
		log.Debug("restore create resp from worker", string(bodyResp))
		restoreCreateResp := new(RestoreContainerResponse)
		json.Unmarshal(bodyResp, &restoreCreateResp)

		fmt.Println("========== received restore create finished", time.Now().UnixNano())
		// update lb address (add new and remove old)
		proxyTarget.LB.Add(restoreCreateResp.ContainerState.Addr)
	}()

	wg.Wait()

	apiAddr := newNode.Addr + "/restore/" + containerState.Name
	restoreReqBodyB, err := json.Marshal(restoreReq)
	body := bytes.NewReader(restoreReqBodyB)

	resp, err := http.Post(
		apiAddr, "application/json", body)

	if err != nil {
		fmt.Println("restore req error: ", err)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println("restore req error: not expected status ", resp.StatusCode)
	}

	defer resp.Body.Close()
	fmt.Println("========== received restore finished", time.Now().UnixNano())

	bodyResp, err := ioutil.ReadAll(resp.Body)
	log.Debug("restore resp from worker", string(bodyResp))
	restoreResp := new(RestoreContainerResponse)
	json.Unmarshal(bodyResp, &restoreResp)

	// update container state
	containerState.Addr = restoreResp.ContainerState.Addr
	containerState.Node = restoreResp.ContainerState.Node
	containerState.Status = restoreResp.ContainerState.Status
	containerState.ContainerID = restoreResp.ContainerState.ContainerID

	return c.String(http.StatusOK, "")
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

	e.POST("/migrate/:name", migrateContainerRequest)

	e.GET("/node/:name", getNodeRequest)
	e.POST("/node/:name", registerNodeRequest)
	e.POST("/node/:name/update", updateNodeRequest)

	e.Logger.Fatal(e.Start(":1323"))
}
