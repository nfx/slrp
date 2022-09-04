package updater

import (
	"fmt"
	"os"
	"runtime"

	"github.com/tj/go-update"
	"github.com/tj/go-update/progress"
	"github.com/tj/go-update/stores/github"
)

func AutoUpdate(version string) {
	m := &update.Manager{
		Command: "slrp",
		Store: &github.Store{
			Owner:   "nfx",
			Repo:    "slrp",
			Version: version,
		},
	}
	releases, err := m.LatestReleases()
	if err != nil {
		panic(err)
	}
	if len(releases) == 0 {
		println("no updates")
		return
	}
	asset := releases[0].FindTarball(runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		fmt.Printf("no binary for %s %s\n", runtime.GOOS, runtime.GOARCH)
		return
	}
	println() // whitespace
	tarball, err := asset.DownloadProxy(progress.Reader)
	if err != nil {
		panic(err)
	}
	err = m.Install(tarball)
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}