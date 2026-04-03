# gocount

A minimal container runtime built in Go. Runs Linux processes in isolated namespaces, like a tiny Docker.

## Features

- Process isolation via Linux namespaces (PID, mount, UTS, network)
- Isolated filesystem using `pivot_root`
- Resource limits via cgroup v2 (CPU, memory)
- Virtual ethernet (`veth`) networking per container
- Container lifecycle management (run, start, stop, remove, inspect)

## Requirements

- Linux (kernel 4.6+ for cgroup v2)
- Root privileges
- `ip` command available (`iproute2`)

## Build

```bash
go build -o gocount .
```

## Usage

### Run a container

```bash
sudo ./gocount run /bin/sh
```

With resource limits:

```bash
sudo ./gocount run --memory 100M --cpu "50000 100000" /bin/sh
```

### List containers

```bash
sudo ./gocount ps
```

### Inspect a container

```bash
sudo ./gocount inspect <container_id>
```

### Stop a container

```bash
sudo ./gocount stop <container_id>
```

### Remove a container

```bash
sudo ./gocount rm <container_id>
```

### Start a stopped container

```bash
sudo ./gocount start <container_id>
```

## How It Works

1. **Run** — spawns a child process with new Linux namespaces
2. **Rootfs** — sets up an isolated filesystem under `/tmp/gocount/<id>/rootfs` using `pivot_root`
3. **Cgroups** — creates a cgroup at `/sys/fs/cgroup/gocount/<id>` and applies CPU/memory limits
4. **Network** — creates a `veth` pair; one end stays on the host, the other goes into the container's network namespace
5. **Metadata** — saves container state as JSON under `/tmp/gocount/<id>.json`

## Project Structure

```
.
├── main.go
├── cmd/
│   ├── root.go       # CLI entrypoint (cobra)
│   ├── run.go        # run & start commands
│   ├── ps.go         # ps command
│   ├── stop.go       # stop & rm commands
│   └── inspect.go    # inspect command
└── internal/
    ├── container/    # container lifecycle & metadata
    ├── cgroups/      # cgroup v2 resource limits
    ├── rootfs/       # rootfs provisioning
    └── network/      # veth pair & network setup
```
