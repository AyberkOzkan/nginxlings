package main

import (
	"context"
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
	modeDoctor
)

type cliOptions struct {
	mode      appMode
	lang      string
	root      string
	state     string
	workspace string
	checkID   string
	noTmux    bool
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
	case modeDoctor:
		return runDoctor(opts, stdout, stderr)
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
	doctor := fs.Bool("doctor", false, "check that the exercises can run")
	fs.StringVar(&opts.checkID, "check", "", "run one exercise check without TUI")
	fs.StringVar(&opts.lang, "lang", "", "initial language code")
	fs.StringVar(&opts.root, "root", "", "asset root containing exercises/ and i18n/")
	fs.StringVar(&opts.state, "state", "", "state file path")
	fs.StringVar(&opts.workspace, "workspace", "", "folder with your own copy of the exercises")
	fs.BoolVar(&opts.noTmux, "no-tmux", false, "run without the terminal pane")

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
	if *doctor {
		if opts.mode != modeTUI {
			return opts, fmt.Errorf("--doctor cannot be combined with --list or --check")
		}
		opts.mode = modeDoctor
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
  --doctor        Check that the tools the exercises need are installed and
                  that Docker answers, and say how to fix what is missing.
  --root <path>   Asset root containing exercises/ and i18n/.
  --state <path>  State file path.
  --workspace <path>
                  Folder with your own copy of the exercises. An installed
                  package uses ~/nginxlings by default.
  --no-tmux       Run without the terminal pane below the app. Without it,
                  nginxlings starts in a tmux session of its own when tmux
                  is installed, with a shell in the exercise folder below.

Environment:
  NGINXLINGS_ASSET_ROOT  Default asset root when --root is not set.
  NGINXLINGS_WORKSPACE   Default exercise folder when --workspace is not set.
  NGINXLINGS_NO_TMUX     Set to any value to run without the terminal pane.
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
	exercises, err := loadWorkspaceExercises(opts)
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

// runDoctor prints the setup check and fails when something is missing.
func runDoctor(opts cliOptions, stdout, stderr io.Writer) int {
	assetRoot, err := ResolveAssetRootWithOptions(opts.root)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	exercises, err := LoadExercises(assetRoot)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	lang := opts.lang
	if lang == "" {
		lang = "en"
	}
	if err := validateLangCode(assetRoot, lang); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	i18n, _ := LoadI18n(assetRoot)
	t := func(key, fallback string) string {
		if i18n == nil {
			return fallback
		}
		return i18n.T(lang, key, fallback)
	}
	report := runSetupCheck(context.Background(), exercises, newSetupEnv())
	fmt.Fprint(stdout, setupText(setupReportLines(report, t)))
	if len(report.problems) > 0 {
		return 1
	}
	return 0
}

// loadCLIExercises loads the shipped exercises without touching a workspace.
func loadCLIExercises(root string) ([]Exercise, error) {
	assetRoot, err := ResolveAssetRootWithOptions(root)
	if err != nil {
		return nil, err
	}
	return LoadExercises(assetRoot)
}

// loadWorkspaceExercises loads the exercises the learner works on, updating
// their workspace first.
func loadWorkspaceExercises(opts cliOptions) ([]Exercise, error) {
	paths, err := ResolveExercisePathsWithOptions(opts.root, opts.workspace)
	if err != nil {
		return nil, err
	}
	if err := syncWorkspace(paths); err != nil {
		return nil, err
	}
	return LoadExercisesFrom(paths.ExercisesDir())
}

func envOverlay(root, state, workspace string) func(string) string {
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
		case envWorkspace:
			if workspace != "" {
				return workspace
			}
		}
		return os.Getenv(key)
	}
}
