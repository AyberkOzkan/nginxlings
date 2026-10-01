package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var version = "dev"

type appMode int

const (
	modeTUI appMode = iota
	modeHelp
	modeVersion
	modeList
	modeCheck
)

type cliOptions struct {
	mode    appMode
	lang    string
	root    string
	state   string
	checkID string
}

type checkRunner func(Exercise) (string, error)
type tuiStarter func(cliOptions) error

func runCLI(args []string, stdout, stderr io.Writer, runner checkRunner, startTUI tuiStarter) int {
	opts, err := parseCLI(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintf(stderr, "Error: %v\n\n", err)
		printUsage(stderr)
		return 2
	}

	switch opts.mode {
	case modeHelp:
		printUsage(stdout)
		return 0
	case modeVersion:
		fmt.Fprintf(stdout, "nginxlings %s\n", version)
		return 0
	case modeList:
		return runList(opts, stdout, stderr)
	case modeCheck:
		return runCheckCommand(opts, stdout, stderr, runner)
	default:
		if err := startTUI(opts); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	}
}

func parseCLI(args []string, output io.Writer) (cliOptions, error) {
	var opts cliOptions

	fs := flag.NewFlagSet("nginxlings", flag.ContinueOnError)
	fs.SetOutput(output)
	help := fs.Bool("help", false, "show help")
	versionFlag := fs.Bool("version", false, "show version")
	list := fs.Bool("list", false, "list exercises")
	fs.StringVar(&opts.checkID, "check", "", "run one exercise check without TUI")
	fs.StringVar(&opts.lang, "lang", "", "initial language code")
	fs.StringVar(&opts.root, "root", "", "asset root containing exercises/ and i18n/")
	fs.StringVar(&opts.state, "state", "", "state file path")

	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if fs.NArg() != 0 {
		return opts, fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	if *help {
		opts.mode = modeHelp
		return opts, nil
	}
	if *versionFlag {
		opts.mode = modeVersion
		return opts, nil
	}
	if *list {
		opts.mode = modeList
	}
	if opts.checkID != "" {
		if opts.mode != modeTUI {
			return opts, fmt.Errorf("--check cannot be combined with --list")
		}
		opts.mode = modeCheck
	}

	return opts, nil
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  nginxlings [flags]

Flags:
  --help          Show this help and exit.
  --version       Show version and exit.
  --lang <code>   Set initial TUI language, for example "en" or "tr".
  --list          List exercises and exit.
  --check <id>    Run one exercise check without starting the TUI.
  --root <path>   Asset root containing exercises/ and i18n/.
  --state <path>  State file path.

Environment:
  NGINXLINGS_ASSET_ROOT  Default asset root when --root is not set.
  NGINXLINGS_STATE_FILE  Default state file when --state is not set.
  NGINXLINGS_STATE_ROOT  Default directory for state.json.
`)
}

func runList(opts cliOptions, stdout, stderr io.Writer) int {
	exercises, err := loadCLIExercises(opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	for _, ex := range exercises {
		fmt.Fprintf(stdout, "%s\t%s\n", ex.ID, ex.DefaultName)
	}
	return 0
}

func runCheckCommand(opts cliOptions, stdout, stderr io.Writer, runner checkRunner) int {
	exercises, err := loadCLIExercises(opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	idx := indexOfExercise(exercises, opts.checkID)
	if idx < 0 {
		fmt.Fprintf(stderr, "Error: exercise %q not found\n", opts.checkID)
		return 1
	}

	output, err := runner(exercises[idx])
	if output != "" {
		fmt.Fprint(stdout, output)
		if !strings.HasSuffix(output, "\n") {
			fmt.Fprintln(stdout)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "Check failed for %s: %v\n", opts.checkID, err)
		return 1
	}
	return 0
}

func loadCLIExercises(root string) ([]Exercise, error) {
	assetRoot, err := ResolveAssetRootWithOptions(root)
	if err != nil {
		return nil, err
	}
	return LoadExercises(assetRoot)
}

func envOverlay(root, state string) func(string) string {
	return func(key string) string {
		switch key {
		case envAssetRoot:
			if root != "" {
				return root
			}
		case envStateFile:
			if state != "" {
				return state
			}
		}
		return os.Getenv(key)
	}
}
