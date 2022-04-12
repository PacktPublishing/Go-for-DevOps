package service

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/coreos/go-systemd/v22/dbus"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/agent/proto"
)

var systemdUnits = ""

func init() {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	systemdUnits = filepath.Join("/home", u.Username, ".config/systemd/user")
	if err := os.MkdirAll(systemdUnits, 0700); err != nil {
		panic(err)
	}
}

var unitTmpl = template.Must(
	template.New("unit").Parse(
		`
[Unit]
Description={{.Desc}}

StartLimitIntervalSec=500
StartLimitBurst=5

[Service]
PrivateUsers=true
PrivateDevices=true
ReadOnlyPaths=/

RootDirectory={{.RootPath}}
ReadWritePaths={{.RootPath}}

SecureBits=noroot

Restart=on-failure
RestartSec=5s

ExecStart={{.BinaryPath}} {{.Args}}

[Install]
WantedBy=multi-user.target
`))

type unitArgs struct {
	Desc       string
	BinaryPath string
	RootPath   string
	Args       string
}

var wufMu sync.Mutex

func writeUnitFile(dbusConn *dbus.Conn, user string, req *pb.InstallReq) error {
	a := unitArgs{
		Desc:       req.Name,
		BinaryPath: filepath.Join("/", req.Binary),
		//BinaryPath: filepath.Join("/home", user, pkgDir, req.Name, req.Binary),
		RootPath: filepath.Join("/home", user, pkgDir, req.Name),
		Args:     strings.Join(req.Args, " "),
	}
	unit := req.Name + ".service"

	p := filepath.Join(systemdUnits, unit)

	f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("could not create systemd unit file: %w", err)
	}

	if err := unitTmpl.Execute(f, a); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Let's only try to reload the daemon one at a time.
	wufMu.Lock()
	defer wufMu.Unlock()
	return dbusConn.Reload()
}

func rmUnitFile(dbusConn *dbus.Conn, user string, req *pb.RemoveReq) error {
	unit := req.Name + ".service"

	p := filepath.Join(systemdUnits, unit)

	if err := os.Remove(p); err != nil {
		if errors.Is(err.(*os.PathError), fs.ErrNotExist) {
			return nil
		}
		return err
	}

	// Let's only try to reload the daemon one at a time.
	wufMu.Lock()
	defer wufMu.Unlock()
	return dbusConn.Reload()
}
