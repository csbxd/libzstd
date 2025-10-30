package main

import (
	"os"
	"os/exec"
	"strings"
)

func main() {
	extArgs := strings.Split(os.Getenv("CCWRAP_EXT_ARGS"), " ")
	args := append(extArgs, os.Args[1:]...)
	cmd := exec.Command(GetCC(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func GetCC() string {
	cc := os.Getenv("CCWRAP_CC")
	if cc != "" {
		return cc
	}
	cc = os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	return cc
}
