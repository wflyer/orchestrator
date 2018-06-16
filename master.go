package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/tcpproxy"
)

var containerStates []*ContainerState

var containerStatesLock = sync.RWMutex{}

func proxyServer(wg sync.WaitGroup) {
	defer wg.Done()
	var p tcpproxy.Proxy
	target := tcpproxy.ToMulti([]string{})
	p.AddRoute(":8081", target) // fallback
	log.Fatal(p.Run())
}

func main() {
	var wg sync.WaitGroup

	containerStates = make([]*ContainerState, 0)
	nodeStates = make([]*NodeState, 0)

	wg.Add(2)
	go apiServer(wg)
	go proxyServer(wg)
	fmt.Println("Started server..")
	wg.Wait()
}
