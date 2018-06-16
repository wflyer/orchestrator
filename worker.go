package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

/*
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
		fmt.Println(state.Name, name)
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
*/

func main() {
	myAddr := os.Getenv("addr")
	nodename := os.Getenv("nodename")
	masterAddr := os.Getenv("masteraddr")

	if myAddr == "" {
		myAddr = "http://127.0.0.1:1324"
	}

	if masterAddr == "" {
		masterAddr = "http://127.0.0.1:1323"
	}
	resp, err := http.PostForm(masterAddr+"/node/"+nodename,
		url.Values{"addr": {myAddr}})

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	/*
		// register to server
		containerStates = make([]*ContainerState, 0)
		e := echo.New()
		e.GET("/", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!")
		})
		e.GET("/containers", listContainerRequest)
		e.POST("/containers/:name", createContainerRequest)
		e.DELETE("/containers/:name", deleteContainerRequest)
		e.Logger.Fatal(e.Start(":1323"))
	*/
}
