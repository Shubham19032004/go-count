package container

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

type Container struct {
	ID      string
	Pid     int
	Command []string
	Status  string
	RootFs  string
	Cgroup  string
}

var Containers = map[string]*Container{}

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
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile("/tmp/gocount/" + f.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read %s: %v\n", f.Name(), err)
			continue
		}
		var c Container
		if err := json.Unmarshal(data, &c); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not parse %s: %v\n", f.Name(), err)
			continue
		}
		containers = append(containers, &c)
	}
	return containers, nil
}
