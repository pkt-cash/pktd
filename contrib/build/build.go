// Copyright (c) 2020 The PKT developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// generic stuff

func die(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

func assertNil(x interface{}, format string, args ...interface{}) {
	if x != nil {
		die(format, args...)
	}
}

type exeF int

const (
	exeEcho       exeF = 1 << iota
	exeCanFail    exeF = 1 << iota
	exeNoRedirect exeF = 1 << iota
)

func exe(flags exeF, name string, arg ...string) (int, string, string) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if flags&exeEcho != 0 {
		fmt.Println(strings.Join(append([]string{name}, arg...), " "))
	}
	cmd := exec.Command(name, arg...)
	if flags&exeNoRedirect == 0 {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()
	ret := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && flags&exeCanFail != 0 {
			ret = ee.ExitCode()
		} else {
			die("exe(%s, %v) -> %v", name, arg, err)
		}
	}
	return ret, stdout.String(), stderr.String()
}

// build stuff

type config struct {
	buildargs []string
	bindir    string
}

func build(name, pkg string, conf *config) {
	if os.Getenv("GOOS") == "windows" {
		name = name + ".exe"
	}
	fmt.Printf("Building %s\n", name)
	args := append([]string{"build", "-o", conf.bindir + "/" + name}, conf.buildargs...)
	args = append(args, pkg)
	exe(exeNoRedirect|exeEcho, "go", args...)
}

func chkdir() {
	info, err := os.Stat("./contrib/build/build.go")
	if err != nil || info.IsDir() {
		die("this script must be invoked from the project root")
	}
}

func buildStr() string {
	exe(0, "git", "update-index", "-q", "--refresh")
	_, id, _ := exe(0, "git", "describe", "--tags", "HEAD")
	id = strings.TrimSpace(id)
	if x, _, _ := exe(exeCanFail, "git", "diff", "--quiet"); x != 0 {
		if os.Getenv("PKT_FAIL_DIRTY") != "" {
			die("Build is dirty, aborting")
		}
		return id + "-dirty"
	}
	return id
}

func ldflags() string {
	return "-w -s -buildid= -X github.com/pkt-cash/pktd/pktconfig/version.appBuild=" + buildStr()
}

func test() {
	fmt.Println("Running tests")
	exe(exeNoRedirect, "go", "test", "-count=1", "-cover", "-covermode=atomic", "-bench=.", "-parallel=1", "-cpu=2", "./...", "-tags=dev")
}

var regex = regexp.MustCompile("[A-Z0-9_]+=.*")

func main() {
	chkdir()
	conf := config{}
	conf.bindir = "./bin"
	conf.buildargs = append(conf.buildargs, "-trimpath", "-ldflags="+ldflags())
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOMAXPROCS", "64")

	assertNil(os.MkdirAll(conf.bindir, 0o755), "mkdir bin")

	for _, a := range os.Args {
		if !regex.MatchString(a) {
			continue
		}
		i := strings.IndexRune(a, '=')
		fmt.Printf("env %s=%s\n", a[0:i], a[i+1:])
		os.Setenv(a[0:i], a[i+1:])
	}

	build("pktd", ".", &conf)
	build("pktwallet", "./pktwallet", &conf)
	build("pktctl", "./cmd/pktctl", &conf)

	if strings.Contains(strings.Join(os.Args, "|"), "--test") {
		os.Setenv("CGO_ENABLED", "1")
		test()
	} else {
		fmt.Println("Pass the --test flag if you want to run the tests as well")
	}
	fmt.Println("Everything looks good, type `./bin/pktwallet --create` to make a wallet")
}
