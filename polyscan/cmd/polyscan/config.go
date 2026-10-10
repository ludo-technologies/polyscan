package main

import (
	"github.com/ludo-technologies/polyscan/polyscan/internal/config"
	"github.com/spf13/cobra"
)

// fileFlags are the flags that choose the configuration file and the files
// analyzed.
type fileFlags struct {
	config       string
	exclude      []string
	includeTests bool
}

func (f *fileFlags) add(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&f.config, "config", "c", "",
		"Configuration file (default: the nearest "+config.FileName+" in the first\n"+
			"path or a directory above it)")
	cmd.Flags().StringSliceVar(&f.exclude, "exclude", nil,
		"Files and directories to leave out (comma-separated or repeated): a glob\n"+
			"without a slash matches a file name or a directory anywhere on the path,\n"+
			"one with a slash matches a path relative to the analyzed directory, with\n"+
			"** for any number of segments (e.g. 'fixtures', 'src/generated/**')")
	cmd.Flags().BoolVar(&f.includeTests, "include-tests", false, "Analyze test files and test code, which are left out by default")
}

// load loads the configuration for an analysis of target and lays the flags
// over it: --exclude adds to the file's patterns, and --include-tests, when
// given, replaces its setting.
func (f *fileFlags) load(cmd *cobra.Command, target string) (*config.Config, error) {
	cfg, err := config.Load(f.config, target)
	if err != nil {
		return nil, err
	}
	cfg.Analysis.Exclude = append(cfg.Analysis.Exclude, f.exclude...)
	override(cmd, "include-tests", &cfg.Analysis.IncludeTests, f.includeTests)
	return cfg, nil
}

// override replaces a setting with the value of its flag when the flag was
// given on the command line.
func override[T any](cmd *cobra.Command, flag string, setting *T, value T) {
	if cmd.Flags().Changed(flag) {
		*setting = value
	}
}
