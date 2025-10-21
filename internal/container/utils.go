package container

import "os"

func EnsureContainerDir() error {
    return os.MkdirAll("/tmp/gocount", 0755)
}
