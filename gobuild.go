package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	rootDir           string
	toolChain         string
	printVersion      bool
	printGo           bool
	updateRate        time.Duration
	versionRegexp     = regexp.MustCompile(`go version devel \+(\w+) .+ \w+\/\w+`)
	errUnknownVersion = errors.New("Unknown version")
)

func init() {
	flag.BoolVar(&printVersion, "v", false, "print go version")
	flag.BoolVar(&printGo, "v", false, "print go executable path")
	flag.StringVar(&rootDir, "r", "/src/go", "go root directory")
	flag.StringVar(&toolChain, "c", "/src/go-linux-amd64-bootstrap/", "go toolchain")
	flag.DurationVar(&updateRate, "t", time.Second*30, "time in seconds to wait between updates")
}

func fatal(err error, description string) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, description, err)
	os.Exit(-1)
}

func getVersion() string {
	cmd := exec.Command(filepath.Join("bin", "go"), "version")
	cmd.Dir = rootDir
	cmd.Stderr = os.Stderr
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	fatal(cmd.Run(), "getVersion")
	found := versionRegexp.FindStringSubmatch(buf.String())
	if len(found) < 2 {
		fatal(errUnknownVersion, "getVersion")
	}
	return found[1]
}

func getCommitHash() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = rootDir
	cmd.Stderr = os.Stderr
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	fatal(cmd.Run(), "git hash")
	return strings.TrimSpace(buf.String())
}

func updateGit() {
	start := time.Now()
	cmd := exec.Command("git", "pull")
	cmd.Dir = rootDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	fatal(cmd.Run(), "git pull")
	elapsed := time.Now().Sub(start)
	fmt.Printf("updated (%s)\n", elapsed)
}

func isUpdateNeeded() bool {
	return getVersion() != getCommitHash()
}

func build() {
	start := time.Now()
	cmd := exec.Command("/bin/bash", "make.bash")
	cmd.Dir = filepath.Join(rootDir, "src")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOROOT_BOOTSTRAP="+toolChain)

	fatal(cmd.Start(), "start")
	fatal(cmd.Wait(), "wait")
	elapsed := time.Now().Sub(start)
	fmt.Println("built in", elapsed)
}

func routine() {
	updateGit()
	if !isUpdateNeeded() {
		fmt.Println("no update needed")
		return
	}
	build()
}

func main() {
	flag.Parse()
	if printVersion {
		fmt.Print(getVersion())
		os.Exit(1)
	}
	fmt.Println("started go builder with rate of", updateRate)
	routine() // first run not in loop
	ticker := time.NewTicker(updateRate)
	for range ticker.C {
		routine()
	}
}
