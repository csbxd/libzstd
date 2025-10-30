package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	ccgo "modernc.org/ccgo/v4/lib"
)

var (
	srcDir    = ""
	patchPath = "libc/patches/0001-fix-zstd-ccgo-build.patch"
)

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	name := filepath.Base(dir)
	if name == "gen" {
		dir = filepath.Dir(filepath.Dir(dir))
	}
	err = os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	srcDir = dir
}

func PrepareDirectory() (tempDir string) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "zstd-build-*")
	if err != nil {
		panic(err)
	}
	run(tempDir, "git", "clone", "-b", "v1.5.7", "--depth=1", "https://github.com/facebook/zstd.git")

	return tempDir
}

func run(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func ApplyPatch(tempDir string) {
	run(filepath.Join(tempDir, "zstd"), "git", "apply", filepath.Join(srcDir, patchPath))
}

func CombineZSTDSources(tempDir string) {
	run(filepath.Join(tempDir, "zstd/build/single_file_libs"),
		"python3", "combine.py",
		"-r", "../../lib", "-x", "legacy/zstd_legacy.h", "-o", "../../../zstd.c", "zstd-in.c")
}

func CompileToGo(zstdCPath string) {
	targets := strings.Split(os.Getenv("ZSTD_GEN_TARGETS"), ";")
	ccs := strings.Split(os.Getenv("ZSTD_GEN_CCS"), ";")
	if targets[0] == "" {
		targets[0] = runtime.GOOS + "_" + runtime.GOARCH
	}

	for i := range targets {
		t := strings.Split(targets[i], "_")
		tOS := t[0]
		tArch := t[1]

		err := os.Setenv("CC", ccs[i])
		if err != nil {
			panic(err)
		}

		switch tOS {
		case "darwin":
			err = ccgo.NewTask(tOS, tArch, []string{
				"ccgo", "--package-name", "libzstd", "-std=c17",
				// libc not implement qsort_r.
				"-ignore-link-errors",
				// __darwin_arm_neon_state64 __darwin_arm_neon_state use __int128, ignore.
				"-ignore-unsupported-alignment",
				zstdCPath, "-o", filepath.Join(srcDir, "zstd_"+targets[i]+".go"),
			}, os.Stdout, os.Stderr, nil).Main()
			if err != nil {
				panic(err)
			}
		case "linux":
			err = ccgo.NewTask(tOS, tArch, []string{
				"ccgo", "--package-name", "libzstd", "-std=c17",
				zstdCPath, "-o", filepath.Join(srcDir, "zstd_"+targets[i]+".go"),
			}, os.Stdout, os.Stderr, nil).Main()
			if err != nil {
				panic(err)
			}
		}
	}

}

func main() {
	dir := PrepareDirectory()
	ApplyPatch(dir)
	CombineZSTDSources(dir)
	CompileToGo(filepath.Join(dir, "zstd.c"))
}
