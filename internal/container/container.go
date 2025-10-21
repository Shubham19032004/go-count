package container

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type Container struct {
	ID      string
	Pid     int
	Command []string
	Status  string
	RootFs  string
}

var Containers = map[string]*Container{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateID() string {
	letters := "abcdefghijklmnopqrstuvwxyz0123456789"
	id := ""
	for i := 0; i < 8; i++ {
		id += string(letters[rand.Intn(len(letters))])
	}
	return id
}

func AddContainer(id string, pid int, command []string, rootfs string) {
	Containers[id] = &Container{
		ID:      id,
		Pid:     pid,
		Command: command,
		Status:  "running",
		RootFs:  rootfs,
	}
	fmt.Println("Container", id, "registered")
}
func SaveContainer(c *Container) error {
	data, _ := json.Marshal(c)
	path := fmt.Sprintf("/tmp/gocount/%s.json", c.ID)
	return os.WriteFile(path, data, 0644)
}

func LoadContainers() ([]*Container, error) {
	var containers []*Container
	files, _ := os.ReadDir("/tmp/gocount")
	for _, f := range files {
		data, _ := os.ReadFile("/tmp/gocount/" + f.Name())
		var c Container
		json.Unmarshal(data, &c)
		containers = append(containers, &c)
	}
	return containers, nil
}
