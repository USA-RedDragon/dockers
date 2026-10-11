package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"gabe565.com/utils/termx"
	"github.com/gabe565/docker-restic/internal/xtrace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	var hasGroupBy, hasStdinFromCommand, hasStdinFilename bool
	stdinName := os.Getenv("RESTIC_HOST")
	var cmd, stdinExt string
	sepIndex := len(args)

outer:
	for i, arg := range args {
		if strings.HasPrefix(arg, "--") {
			key, _, _ := strings.Cut(arg, "=")
			switch {
			case key == "--group-by":
				hasGroupBy = true
			case key == "--stdin-from-command":
				hasStdinFromCommand = true
			case key == "--stdin-filename":
				hasStdinFilename = true
			case arg == "--":
				sepIndex = i
				subCmd := args[i+1:]
				if len(subCmd) >= 2 && filepath.Base(subCmd[0]) == "dumpdb" {
					stdinExt = ".sql"
					switch subCmd[1] {
					case "cnpg", "mongodb":
						stdinExt = ".dmp"
					case "sqlite":
						if len(subCmd) >= 3 {
							stdinName = subCmd[2]
							ext := filepath.Ext(stdinName)
							switch ext {
							case ".db", ".sqlite3", ".sqlite":
								stdinName = strings.TrimSuffix(stdinName, ext)
							}
						}
					}
				}
				break outer
			}
		} else if cmd == "" {
			cmd = arg
		}
	}

	finalArgs := slices.Grow(slices.Clone(args[:sepIndex]), 3)

	switch cmd {
	case "forget":
		if !termx.IsTerminal(os.Stdout) {
			finalArgs = append(finalArgs, "--compact")
		}
	case "backup", "snapshots":
	default:
		return execRestic(args)
	}

	if !hasGroupBy {
		if groupBy := os.Getenv("RESTIC_GROUP_BY"); groupBy != "" {
			finalArgs = append(finalArgs, "--group-by="+groupBy)
		}
	}

	if hasStdinFromCommand && !hasStdinFilename && stdinName != "" {
		finalArgs = append(finalArgs, "--stdin-filename="+stdinName+stdinExt)
	}

	finalArgs = append(finalArgs, args[sepIndex:]...)
	return execRestic(finalArgs)
}

func execRestic(args []string) error {
	path := "/usr/bin/restic"
	argv := append([]string{path}, args...)

	xtrace.XTrace(os.Stderr, argv)

	if dryRunStr := os.Getenv("RESTIC_WRAPPER_DRY_RUN"); dryRunStr != "" {
		if dryRun, _ := strconv.ParseBool(dryRunStr); dryRun {
			return nil
		}
	}

	return syscall.Exec(path, argv, os.Environ())
}
