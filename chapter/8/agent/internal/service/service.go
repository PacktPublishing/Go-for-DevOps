// Package service contains the Agent that provides control access to the system
// and system stats.
package service

import (
	"archive/zip"
	"bytes"
	"context"
	"expvar"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	linuxproc "github.com/c9s/goprocinfo/linux"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/agent/proto"
)

const (
	// pkgDir is the directory in the Agent user's home where we are installing and
	// running packages. A more secure version would be to have the agent do this
	// in individual user directories that match some user on all machines. However
	// this is for illustration purposes only.
	pkgDir     = "sa/packages/"
	serviceExt = ".service"
)

// Agent provides a system agent service that runs a gRPC service for doing
// application installs and an HTTP service for relaying stats.
type Agent struct {
	pb.UnimplementedAgentServer

	dbus *dbus.Conn
	user string

	// mu protects locks.
	mu    sync.Mutex
	locks map[string]*sync.Mutex

	cpuData, memData atomic.Value
}

// New creates a new Agent instance.
func New() (*Agent, error) {
	conn, err := dbus.NewUserConnection()
	if err != nil {
		return nil, fmt.Errorf("problem connecting to systemd: %w", err)
	}
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return &Agent{
		dbus:  conn,
		user:  u.Username,
		locks: map[string]*sync.Mutex{},
	}, nil
}

// Start starts the agent. As the agent is not intended to ever stop, this has
// no Stop(). This blocks unless there is a problem.
func (a *Agent) Start() error {
	var sockAddr = filepath.Join("/home", a.user, "/sa/socket/sa.sock")
	if err := os.MkdirAll(filepath.Dir(sockAddr), 0700); err != nil {
		return fmt.Errorf("could not create socket dir path: %w", err)
	}
	// Remove old socket file if it exists.
	os.Remove(sockAddr)

	if err := a.perfLoop(); err != nil {
		return err
	}

	l, err := net.Listen("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("could not connect to socket: %w", err)
	}

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterAgentServer(grpcServer, a)
	return grpcServer.Serve(l)
}

// Install implements our gRPC Install RPC.
func (a *Agent) Install(ctx context.Context, req *pb.InstallReq) (*pb.InstallResp, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	a.lock(req.Name)
	defer a.unlock(req.Name, false)

	loc, err := a.unpack(req.Name, req.Package)
	if err != nil {
		return nil, err
	}

	if err := a.migrate(req, loc); err != nil {
		return nil, err
	}

	if err := a.startProgram(ctx, req.Name); err != nil {
		return nil, err
	}
	return &pb.InstallResp{}, nil
}

func (a *Agent) Remove(ctx context.Context, req *pb.RemoveReq) (*pb.RemoveResp, error) {
	a.lock(req.Name)
	defer a.unlock(req.Name, true)

	if err := a.stopProgram(ctx, req.Name); err != nil {
		return nil, err
	}

	if err := rmUnitFile(a.dbus, a.user, req); err != nil {
		return nil, err
	}
	return &pb.RemoveResp{}, nil
}

// lock locks a named mutex.
func (a *Agent) lock(name string) {
	a.mu.Lock()
	v, ok := a.locks[name]
	if !ok {
		v = &sync.Mutex{}
		a.locks[name] = v
	}
	a.mu.Unlock()

	v.Lock()
}

// unlock unlocks a named mutex.
func (a *Agent) unlock(name string, del bool) {
	a.mu.Lock()
	v, ok := a.locks[name]
	if !ok {
		return
	}
	if del {
		delete(a.locks, name)
	}
	a.mu.Unlock()
	v.Unlock()
}

// unpack unpacks a zipfile and stores in in a temporary directory that is returned.
func (a *Agent) unpack(name string, zipFile []byte) (string, error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("sa_install_%s_*", name))
	if err != nil {
		return "", err
	}
	r, err := zip.NewReader(bytes.NewReader(zipFile), int64(len(zipFile)))
	if err != nil {
		return "", err
	}

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		if err := a.writeFile(f, dir); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// writeFile writes a zip file under the root directory dir.
func (a *Agent) writeFile(z *zip.File, dir string) error {
	if z.FileInfo().IsDir() {
		err := os.Mkdir(
			filepath.Join(dir, filepath.FromSlash(z.Name)),
			z.Mode(),
		)
		return err
	}

	rc, err := z.Open()
	if err != nil {
		return fmt.Errorf("could not open file %q: %w", z.Name, err)
	}
	defer rc.Close()

	nf, err := os.OpenFile(
		filepath.Join(dir, filepath.FromSlash(z.Name)),
		os.O_CREATE|os.O_WRONLY,
		z.Mode(),
	)
	if err != nil {
		return fmt.Errorf("could not open file in temp diretory: %w", err)
	}
	defer nf.Close()

	_, err = io.Copy(nf, rc)
	if err != nil {
		return fmt.Errorf("file copy error: %w", err)
	}
	return nil
}

// migrate shuts down any existing job that is running and migrates our files
// from the temp location to the final location.
func (a *Agent) migrate(req *pb.InstallReq, loc string) error {
	units, err := a.dbus.ListUnitsByNames([]string{req.Name + serviceExt})
	if err == nil && units[0].JobId != 0 {
		result := make(chan string, 1)
		_, err := a.dbus.StopUnit(req.Name+serviceExt, "replace", result)
		if err != nil {
			return fmt.Errorf("migate could not stop the service: %w", err)
		}
		switch v := <-result; v {
		case "done":
		default:
			return fmt.Errorf("systemd StopUnit() returned %q", v)
		}
		//a.conn.KillUnit(name, 15)
	}
	if err := writeUnitFile(a.dbus, a.user, req); err != nil {
		return fmt.Errorf("could not write the unit file: %w", err)
	}

	p := filepath.Join("/home", a.user, pkgDir)
	if _, err := os.Stat(p); err == nil {
		os.RemoveAll(p)
	}
	if err := os.Rename(loc, p); err != nil {
		return err
	}
	return nil
}

// startProgram starts our program under systemd.
func (a *Agent) startProgram(ctx context.Context, name string) error {
	// EnableUnitFiles(files []string, runtime bool, force bool) (bool, []EnableUnitFileChange, error)
	result := make(chan string, 1)
	id, err := a.dbus.StartUnit(name+serviceExt, "replace", result)
	if err != nil {
		return fmt.Errorf("could not start the unit: %w", err)
	}
	switch v := <-result; v {
	case "done":
		log.Printf("new service(%s) is done: %v", name+serviceExt, id)
	default:
		return fmt.Errorf("systemd StartUnit() returned %q", v)
	}

	time.Sleep(30 * time.Second)
	statuses, err := a.dbus.ListUnitsByNames([]string{name + serviceExt})
	if err != nil {
		return fmt.Errorf("could not find unit after start: %s", err)
	}
	if len(statuses) != 1 {
		return fmt.Errorf("could not find unit after start")
	}
	status := statuses[0]
	switch {
	case status.ActiveState != "active":
		return fmt.Errorf("program is not in active state")
	case status.SubState != "running":
		return fmt.Errorf("program is not in running state")
	case status.LoadState != "loaded":
		return fmt.Errorf("program is not in loaded state")
	}
	return nil
}

// stopProgram stops a program under systemd.
func (a *Agent) stopProgram(ctx context.Context, name string) error {
	result := make(chan string, 1)
	_, err := a.dbus.StopUnit(name+serviceExt, "replace", result)
	if err != nil {
		return fmt.Errorf("could not stop the service: %w", err)
	}
	switch v := <-result; v {
	case "done":
	default:
		return fmt.Errorf("systemd StopUnit() returned %q", v)
	}
	return nil
}

// collectCPU collects our CPU stats and stores them in .cpuData.
func (a *Agent) collectCPU(resolution int32) error {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		return err
	}
	v := &pb.CPUPerfs{
		ResolutionSecs: resolution,
		UnixTimeNano:   time.Now().UnixNano(),
	}
	for _, p := range stat.CPUStats {
		c := &pb.CPUPerf{
			Id:     p.Id,
			User:   int32(p.User),
			System: int32(p.System),
			Idle:   int32(p.Idle),
			IoWait: int32(p.IOWait),
			Irq:    int32(p.IRQ),
		}
		v.Cpu = append(v.Cpu, c)
	}
	a.cpuData.Store(v)
	return nil
}

// collectMem collects our memory stats and stores them in .memData.
func (a *Agent) collectMem(resolution int32) error {
	mem, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		return err
	}
	a.memData.Store(
		&pb.MemPerf{
			ResolutionSecs: resolution,
			UnixTimeNano:   time.Now().UnixNano(),
			Total:          int32(mem.MemTotal),
			Free:           int32(mem.MemFree),
			Avail:          int32(mem.MemAvailable),
		},
	)
	return nil
}

// perfLoop grabs data every 10 seconds + gather time and stores it.
// It also does all registration of these variables with expvar.
// This should only be called once on systemAgent start.
func (a *Agent) perfLoop() error {
	const resolutionSecs = 10

	if err := a.collectCPU(resolutionSecs); err != nil {
		return err
	}
	if err := a.collectMem(resolutionSecs); err != nil {
		return err
	}

	expvar.Publish(
		"system-cpu",
		expvar.Func(
			func() interface{} {
				return a.cpuData.Load().(*pb.CPUPerfs)
			},
		),
	)
	expvar.Publish(
		"system-mem",
		expvar.Func(
			func() interface{} {
				return a.memData.Load().(*pb.MemPerf)
			},
		),
	)

	go func() {
		for {
			time.Sleep(resolutionSecs * time.Second)
			if err := a.collectCPU(resolutionSecs); err != nil {
				log.Println(err)
			}
			if err := a.collectMem(resolutionSecs); err != nil {
				log.Println(err)
			}
		}
	}()
	return nil
}
