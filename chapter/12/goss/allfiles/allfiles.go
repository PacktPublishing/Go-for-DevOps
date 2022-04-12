//go:build linux || darwin

/*
	^
	|
If you haven't see the above line, this is called a build constraint. This will
only build for linux or darwin (osx). You can add other build contstraints for architectures or make it so that there are multiple build constraints.
You may also use this to have only certain files that are compiled for certain platforms. This allows you to compile in OS/Arch specific instructions
for each platform you support. This is how the standard library "sys" package works, there is a different "sys" for each platform.

We are restricting this due to the need to read file permissions that we are only sure of on linux and darwin.
*/

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/aelsabbahy/goss"
	"github.com/aelsabbahy/goss/resource"
	"gopkg.in/yaml.v2"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("must have single argument, the root path to walk")
		os.Exit(1)
	}
	if !strings.HasPrefix(os.Args[1], "/") {
		fmt.Println("path argument must be fully qualified")
		os.Exit(1)
	}
	conf := goss.GossConfig{
		Files: resource.FileMap{},
	}

	input := make(chan fileEntry, 1)

	go func() {
		defer close(input)
		err := filepath.WalkDir(
			os.Args[1],
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				input <- fileEntry{p: path, info: info}
				return nil
			},
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	limit := make(chan struct{}, 1000)
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for fe := range input {
		fe := fe
		limit <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-limit }()

			rsc, err := addFileEntries(fe)
			if err != nil {
				log.Println("file could not be added: ", err)
				return
			}
			mu.Lock()
			conf.Files[fe.p] = rsc
			mu.Unlock()
		}()
	}
	wg.Wait()

	d, err := yaml.Marshal(conf)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(d))
}

type fileEntry struct {
	p    string
	info os.FileInfo
}

func addFileEntries(fe fileEntry) (*resource.File, error) {
	u, g := mustUserGroup(fe)

	h, err := doSHA256(fe.p)
	if err != nil {
		return nil, err
	}

	return &resource.File{
		Path:     fe.p,
		Exists:   true,
		Mode:     fmt.Sprintf("%04o", fe.info.Mode()),
		Owner:    u,
		Group:    g,
		Filetype: "file",
		Sha256:   h,
	}, nil
}

func mustUserGroup(fe fileEntry) (string, string) {
	stat := fe.info.Sys().(*syscall.Stat_t)
	uid := stat.Uid
	gid := stat.Gid
	u := strconv.FormatUint(uint64(uid), 10)
	g := strconv.FormatUint(uint64(gid), 10)
	usr, err := user.LookupId(u)
	if err != nil {
		panic(fmt.Sprintf("file(%s) had UID(%d) we couldn't fine: %s", fe.p, uid, err))
	}
	group, err := user.LookupGroupId(g)
	if err != nil {
		panic(fmt.Sprintf("file(%s) had GID(%d) we couldn't fine: %s", fe.p, gid, err))
	}
	return usr.Username, group.Name
}

func doSHA256(p string) (string, error) {
	dst := sha256.New()

	src, err := os.Open(p)
	if err != nil {
		return "", fmt.Errorf("could not open file: %s", p)
	}
	defer src.Close()

	if _, err := io.Copy(dst, src); err != nil {
		panic(fmt.Sprintf("problem doing an SHA256(%s): %s", p, err))
	}

	return fmt.Sprintf("%x", dst.Sum(nil)), nil
}
