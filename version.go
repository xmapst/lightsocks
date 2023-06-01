package lightsocks

import (
	"fmt"
	"runtime"

	"github.com/common-nighthawk/go-figure"
)

var (
	Name      = "LightSocks"
	Version   string
	GoVersion = runtime.Version()
	GoOs      = runtime.GOOS
	GoArch    = runtime.GOARCH
	GitUrl    = "https://github.com/xmapst/lightsocks.git"
	GitBranch string
	GitCommit string
	BuildTime string
	title     = figure.NewFigure(Name, "doom", true).String()
)

const header = `Version: %s
GoVer: %s
GoOs: %s
GoArch: %s
GitUrl: %s
GitBranch: %s
GitCommit: %s
BuildTime: %s
`

func VersionIfo() string {
	return fmt.Sprintf(header, Version, GoVersion, GoOs, GoArch, GitUrl, GitBranch, GitCommit, BuildTime)
}

func PrintHeadInfo() {
	fmt.Println(title)
	fmt.Println(VersionIfo())
}
