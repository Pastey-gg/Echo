package state

import (
	"os"
	"runtime/debug"

	"github.com/EvieePy/Echo/logger"
)

type VersionInfo struct {
	Version    string
	CommitHash string
	CommitTime string
}

var (
	commitHash = ""
	commitTime = ""
)

func getVersion(logger *logger.Logger) string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		logger.Fatal("Unable to find version info.")
	}

	return string(data)
}

func NewVersionInfo(logger *logger.Logger) VersionInfo {
	var verInfo VersionInfo
	verInfo.Version = getVersion(logger)
	verInfo.CommitHash = commitHash
	verInfo.CommitTime = commitTime

	info, ok := debug.ReadBuildInfo()
	if !ok {
		logger.Warnf("Unable to fetch build information.")
		return verInfo
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if verInfo.CommitHash == "" {
				verInfo.CommitHash = setting.Value
			}
		case "vcs.time":
			if verInfo.CommitTime == "" {
				verInfo.CommitTime = setting.Value
			}
		}
	}

	return verInfo
}
