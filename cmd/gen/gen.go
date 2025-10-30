package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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

func env(name, deflt string) (r string) {
	r = deflt
	if s := os.Getenv(name); s != "" {
		r = s
	}
	return r
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
	singleDir := filepath.Join(tempDir, "zstd/build/single_file_libs")
	run(singleDir,
		"python3", "combine.py",
		"-r", "../../lib", "-x", "legacy/zstd_legacy.h", "-o", "../../../zstd.c", "zstd-in.c")

}

func CompileToGo(zstdCPath string) {
	goarch := env("TARGET_GOARCH", env("GOARCH", runtime.GOARCH))
	goos := env("TARGET_GOOS", env("GOOS", runtime.GOOS))

	switch goos {
	case "darwin":
		err := ccgo.NewTask(goos, goarch, []string{
			"ccgo", "--package-name", "libzstd", "-std=c17",
			// libc not implement qsort_r.
			"-ignore-link-errors",
			// __darwin_arm_neon_state64 __darwin_arm_neon_state use __int128, ignore.
			"-ignore-unsupported-alignment",
			zstdCPath, "-o", filepath.Join(srcDir, "zstd_"+goos+"_"+goarch+".go"),
		}, os.Stdout, os.Stderr, nil).Main()
		if err != nil {
			panic(err)
		}
	case "linux":
		err := ccgo.NewTask(goos, goarch, []string{
			"ccgo", "--package-name", "libzstd", "-std=c17",
			zstdCPath, "-o", filepath.Join(srcDir, "zstd_"+goos+"_"+goarch+".go"),
		}, os.Stdout, os.Stderr, nil).Main()
		if err != nil {
			panic(err)
		}
	case "windows":
		err := ccgo.NewTask(goos, goarch, []string{
			"ccgo", "--package-name", "libzstd", "-std=c17",
			// libc not implement qsort_s clock.
			"-ignore-link-errors",
			// ignore some builtin func
			"-D_IMMINTRIN_H_INCLUDED", "-D_FMA4INTRIN_H_INCLUDED", "-D_XOPMMINTRIN_H_INCLUDED",
			zstdCPath, "-o", filepath.Join(srcDir, "zstd_"+goos+"_"+goarch+".go"),
		}, os.Stdout, os.Stderr, nil).Main()
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	dir := PrepareDirectory()
	ApplyPatch(dir)
	CombineZSTDSources(dir)
	CompileToGo(filepath.Join(dir, "zstd.c"))
}
