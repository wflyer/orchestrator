package main

import (
	"fmt"

	"github.com/satori/go.uuid"
)

func main() {
	var err error

	err = removeContainer("counter-server")
	if err != nil {
		fmt.Println(err)
	}

	containerID, err := runContainer("counter-server", "counter", []string{"/app/server"}, []string{"8080:8080/tcp"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(containerID)

	uuidstr := uuid.NewV4()
	checkpointID := fmt.Sprintf("counter-server-%s", uuidstr)
	checkpointDir := "/tmp/" + checkpointID
	fmt.Println(checkpointID)

	err = checkpointContainer("counter-server", checkpointID, checkpointDir)
	if err != nil {
		fmt.Println(err)
	}

	err = restoreCointainer("counter-server", checkpointID, checkpointDir)
	if err != nil {
		fmt.Println(err)
	}
}
