package main

import (
	"sort"
	"sync"

	"github.com/pkg/errors"
)

func (n *NodeState) ContainerCnt() int {
	cnt := 0
	for _, containerState := range containerStates {
		if containerState.Node == n.Name {
			cnt++
		}
	}
	return cnt
}

func (n *NodeState) ContainerStates() []*ContainerState {
	states := make([]*ContainerState, 0)

	for _, containerState := range containerStates {
		if containerState.Node == n.Name {
			states = append(states, containerState)
		}
	}
	return states
}

var nodeStates NodeStates

func (a NodeStates) Len() int           { return len(a) }
func (a NodeStates) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a NodeStates) Less(i, j int) bool { return a[i].ContainerCnt() < a[j].ContainerCnt() }

var nodeStatesLock = sync.RWMutex{}

// FindAllocNode finds appropriate node to host the container
// except: optional (can be empty)
func FindAllocNode(except string) (*NodeState, error) {

	// find lowest allocated
	nodeStatesLock.Lock()
	defer nodeStatesLock.Unlock()
	sort.Sort(nodeStates)

	for i := len(nodeStates) - 1; i >= 0; i-- {
		if nodeStates[i].Name != except {
			return nodeStates[i], nil
		}
	}
	return nil, errors.New("No suitable node available.")
}
