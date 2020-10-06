// Package commands implements subcommands of cos_gpu_installer.
package commands

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"flag"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/installer"
	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/internal/signing"
	"cos.googlesource.com/cos/tools.git/src/pkg/cos"
	"cos.googlesource.com/cos/tools.git/src/pkg/modules"

	log "github.com/golang/glog"
	"github.com/google/subcommands"
	"github.com/pkg/errors"
)

const (
	hostRootPath    = "/root"
	kernelSrcDir    = "/build/usr/src/linux"
	kernelHeaderDir = "/build/usr/src/linux-headers"
	toolchainPkgDir = "/build/cos-tools"
)

// InstallCommand is the subcommand to install GPU drivers.
type InstallCommand struct {
	driverVersion    string
	hostInstallDir   string
	unsignedDriver   bool
	internalDownload bool
	debug            bool
}

// Name implements subcommands.Command.Name.
func (*InstallCommand) Name() string { return "install" }

// Synopsis implements subcommands.Command.Synopsis.
func (*InstallCommand) Synopsis() string { return "Install GPU drivers." }

// Usage implements subcommands.Command.Usage.
func (*InstallCommand) Usage() string { return "install [-dir <filepath>]\n" }

// SetFlags implements subcommands.Command.SetFlags.
func (c *InstallCommand) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.driverVersion, "version", "",
		"The GPU driver verion to install. It will install the default GPU driver if the flag is not set explicitly.")
	f.StringVar(&c.hostInstallDir, "host-dir", "",
		"Host directory that GPU drivers should be installed to. "+
		"It tries to read from the env NVIDIA_INSTALL_DIR_HOST if the flag is not set explicitly.")
	f.BoolVar(&c.unsignedDriver, "allow-unsigned-driver", false,
		"Whether to allow load unsigned GPU drivers. "+
			"If this flag is set to true, module signing security features must be disabled on the host for driver installation to succeed. "+
			"This flag is only for debugging.")
	// TODO(mikewu): change this flag to a bucket prefix string.
	f.BoolVar(&c.internalDownload, "internal-download", false,
		"Whether to try to download files from Google internal server. This is only useful for internal developing.")
	f.BoolVar(&c.debug, "debug", false,
		"Enable debug mode.")
}

// Execute implements subcommands.Command.Execute.
func (c *InstallCommand) Execute(ctx context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	envReader, err := cos.NewEnvReader(hostRootPath)
	if err != nil {
		c.logError(errors.Wrapf(err, "failed to create envReader with host root path %s", hostRootPath))
		return subcommands.ExitFailure
	}

	log.Infof("Running on COS build id %s", envReader.BuildNumber())

	downloader := cos.NewGCSDownloader(envReader, c.internalDownload)
	if c.driverVersion == "" {
		defaultVersion, err := installer.GetDefaultGPUDriverVersion(downloader)
		if err != nil {
			c.logError(errors.Wrap(err, "failed to get default driver version"))
			return subcommands.ExitFailure
		}
		c.driverVersion = defaultVersion
	}
	log.Infof("Installing GPU driver version %s", c.driverVersion)

	if c.unsignedDriver {
		kernelCmdline, err := ioutil.ReadFile("/proc/cmdline")
		if err != nil {
			c.logError(fmt.Errorf("failed to read kernel command line: %v", err))
		}
		if !cos.CheckKernelModuleSigning(string(kernelCmdline)) {
			log.Warning("Current kernel command line does not support unsigned kernel modules. Not enforcing kernel module signing may cause installation fail.")
		}
	}

	// Read value from env NVIDIA_INSTALL_DIR_HOST if the flag is not set. This is to be compatible with old interface.
	if c.hostInstallDir == "" {
		c.hostInstallDir = os.Getenv("NVIDIA_INSTALL_DIR_HOST")
	}
	hostInstallDir := filepath.Join(hostRootPath, c.hostInstallDir)
	cacher := installer.NewCacher(hostInstallDir, envReader.BuildNumber(), c.driverVersion)
	if isCached, err := cacher.IsCached(); isCached && err == nil {
		log.Info("Found cached version, NOT building the drivers.")
		if err := installer.ConfigureCachedInstalltion(hostInstallDir, !c.unsignedDriver); err != nil {
			c.logError(errors.Wrap(err, "failed to configure cached installation"))
			return subcommands.ExitFailure
		}
		if err := installer.VerifyDriverInstallation(); err != nil {
			c.logError(errors.Wrap(err, "failed to verify GPU driver installation"))
			return subcommands.ExitFailure
		}
		if err := modules.UpdateHostLdCache(hostRootPath, filepath.Join(c.hostInstallDir, "lib64")); err != nil {
			c.logError(errors.Wrap(err, "failed to update host ld cache"))
			return subcommands.ExitFailure
		}
		return subcommands.ExitSuccess
	}

	log.Info("Did not find cached version, installing the drivers...")
	if err := installDriver(c, cacher, envReader, downloader); err != nil {
		c.logError(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func installDriver(c *InstallCommand, cacher *installer.Cacher, envReader *cos.EnvReader, downloader *cos.GCSDownloader) error {
	callback, err := installer.ConfigureDriverInstallationDirs(filepath.Join(hostRootPath, c.hostInstallDir), envReader.KernelRelease())
	if err != nil {
		return errors.Wrap(err, "failed to configure GPU driver installation dirs")
	}
	defer func() { callback <- 0 }()

	if !c.unsignedDriver {
		if err := signing.DownloadDriverSignatures(downloader, c.driverVersion); err != nil {
			return errors.Wrap(err, "failed to download driver signature")
		}
	}

	installerFile, err := installer.DownloadDriverInstaller(
		c.driverVersion, envReader.Milestone(), envReader.BuildNumber())
	if err != nil {
		return errors.Wrap(err, "failed to download GPU driver installer")
	}

	if err := cos.SetCompilationEnv(downloader); err != nil {
		return errors.Wrap(err, "failed to set compilation environment variables")
	}
	if err := cos.InstallCrossToolchain(downloader, toolchainPkgDir); err != nil {
		return errors.Wrap(err, "failed to install toolchain")
	}

	if err := installer.RunDriverInstaller(installerFile, !c.unsignedDriver); err != nil {
		return errors.Wrap(err, "failed to run GPU driver installer")
	}
	if err := cacher.Cache(); err != nil {
		return errors.Wrap(err, "failed to cache installation")
	}
	if err := installer.VerifyDriverInstallation(); err != nil {
		return errors.Wrap(err, "failed to verify installation")
	}
	if err := modules.UpdateHostLdCache(hostRootPath, filepath.Join(c.hostInstallDir, "lib64")); err != nil {
		return errors.Wrap(err, "failed to update host ld cache")
	}
	log.Info("Finished installing the drivers.")
	return nil
}

func (c *InstallCommand) logError(err error) {
	if c.debug {
		log.Errorf("%+v", err)
	} else {
		log.Errorf("%v", err)
	}
}
