package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"bytes"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
)

var syncing sync.RWMutex

var imageName = "counter"
var imageCmd = []string{"/app/server"}
var exposedPorts = []string{"8080:8080/tcp"}

func syncWithMaster() {

	syncing.RLock()
	defer syncing.RUnlock()

	resp, err := http.Get(masterAddr + "/node/" + nodename)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Debug("sync from server:", string(body))
	nodeResp := NodeResponse{}
	json.Unmarshal(body, &nodeResp)

	updatedCStates := make([]*ContainerState, 0)

	for _, cState := range nodeResp.ContainerStates {
		if cState.Status == "pending" {
			// run container
			// 1. execute run container
			cID, err := runContainer(cState.Name, imageName, imageCmd, exposedPorts)
			if err != nil {
				log.Error("error running container: ", err)
				continue
			}
			newCState := &ContainerState{
				Name:        cState.Name,
				ContainerID: cID,
				Node:        nodename,
				Status:      "running",
			}
			updatedCStates = append(updatedCStates, newCState)
		}
	}
	// report to server
	if len(updatedCStates) > 0 {
		updatedReq := NodeUpdateRequest{
			Name: nodename,
			UpdatedContainerStates: updatedCStates,
		}
		bodyB, err := json.Marshal(updatedReq)
		log.Debug("sync to master:", string(bodyB))
		if err != nil {
			panic(err)
		}

		body := bytes.NewReader(bodyB)
		resp, err := http.Post(
			masterAddr+"/node/"+nodename+"/update", "application/json", body)

		if err != nil {
			fmt.Println("update error: ", err)
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Println("update error: not expected status ", resp.StatusCode)
		}
	}

	time.Sleep(time.Duration(10) * time.Second)
	go syncWithMaster()
}

var (
	myAddr     string
	nodename   string
	masterAddr string
)

func workerServer() {
	e := echo.New()
	/*
		e.GET("/", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!")
		})
		e.GET("/containers", listContainerRequest)
		e.POST("/containers/:name", createContainerRequest)
		e.DELETE("/containers/:name", deleteContainerRequest)
	*/
	e.Logger.Fatal(e.Start(":1324"))
}

func main() {
	///containerStates = make([]*ContainerState, 0)
	log.SetLevel(log.DEBUG)

	myAddr = os.Getenv("addr")
	nodename = os.Getenv("nodename")
	masterAddr = os.Getenv("masteraddr")

	if myAddr == "" {
		myAddr = "http://127.0.0.1:1324"
	}

	if masterAddr == "" {
		masterAddr = "http://127.0.0.1:1323"
	}

	// Register to server
	resp, err := http.PostForm(masterAddr+"/node/"+nodename,
		url.Values{"addr": {myAddr}})

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	// apiserver to receive req from master
	go workerServer()

	go syncWithMaster()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
