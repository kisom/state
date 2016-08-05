package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
)

var (
	changesOnly = []string{"--state-output", "changes"}
	fullOutput  = []string{"--state-output", "full"}
	local       = "--local"
	logDebug    = []string{"-l", "debug"}
	logQuiet    = []string{"-l", "warning"}
	noColour    = "--no-color"
	outFile     = "--out-file"
	salt        = "salt"
	saltCall    = "salt-call"
)

func usage(w io.Writer) {
	fmt.Fprintf(w, `usage: state [-dgm] action [args...]
state is a wrapper for commonly used salt functions. It defaults to
using salt-call --local for local state testing and management.

Actions:
	sls		Apply a salt state. This requires at least one argument
			that is the state to apply.
	up		Run a highstate.
	highstate	Run a highstate.
	sync		Sync Salt and Pillar.
	clear		Clear the minion cache.
	
Flags:
	-c	Turn on coloured output.
	-d	Use the DEBUG level of logging in the Salt binary.
	-f	Also write output to the specified file.
	-g	The Salt command should be global (e.g. use salt instead
		of salt-call); the first argument after the action should
		be a target spec; implies -m.
	-m	Use the salt master (e.g. no --local).
	-q	Quiet mode: only show warning and error log messages.
	-v	Show full Salt output, instead of just changes.
`)
}

const localOnly = true

func buildCommand(args []string) (cmd *exec.Cmd, err error) {
	cmd = &exec.Cmd{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	cmd.Path, err = exec.LookPath(args[0])
	if err != nil {
		return nil, err
	}

	cmd.Args = args
	return cmd, nil
}

func fatalf(err error, format string, args ...interface{}) {
	if format == "" {
		format = fmt.Sprintf("%s\n", err)
	} else {
		format += fmt.Sprintf(" (err = %s)\n", err)
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func sync(arglist, argv []string) {
	pillarSync := append(arglist, "saltutil.refresh_pillar")
	pillarSync = append(pillarSync, argv...)
	cmd, err := buildCommand(pillarSync)
	if err != nil {
		fatalf(err, "failed to find %s (is salt installed?)", pillarSync[0])
	}

	err = cmd.Run()
	if err != nil {
		fatalf(err, "failed to refresh pillar")
	}

	saltSync := append(arglist, "saltutil.sync_all")
	saltSync = append(saltSync, argv...)
	cmd, err = buildCommand(pillarSync)
	if err != nil {
		fatalf(err, "failed to find %s (is salt installed?)", saltSync[0])
	}

	err = cmd.Run()
	if err != nil {
		fatalf(err, "failed to sync salt")
	}
	os.Exit(0)
}

func main() {
	var (
		// Flags.
		colour    bool
		debug     bool
		full      bool
		global    bool
		quiet     bool
		useMaster bool

		// Options.
		outPath string

		// Argument handling.
		arglist []string
		argc    int
		argv    []string
		action  string
		target  string
	)

	flag.BoolVar(&colour, "c", false, "Turn on coloured output.")
	flag.BoolVar(&debug, "d", false, "Turn on debug logging.")
	flag.StringVar(&outPath, "f", "", "Also write logs to the named file.")
	flag.BoolVar(&global, "g", false, "Global salt command.")
	flag.BoolVar(&useMaster, "m", false, "Use the Salt master.")
	flag.BoolVar(&quiet, "q", false, "Only show warnings and errors in logs.")
	flag.BoolVar(&full, "v", false, "Show full output.")
	flag.Parse()

	argc = flag.NArg()
	argv = flag.Args()
	if argc == 0 {
		usage(os.Stdout)
		return
	}

	action = argv[0]
	argv = argv[1:]
	argc--

	if global {
		if argc == 0 {
			usage(os.Stderr)
			os.Exit(1)
		}
		target = argv[0]
		argv = argv[1:]
		argc--
	}

	if global {
		arglist = append(arglist, salt)
		arglist = append(arglist, target)
		useMaster = true
	} else {
		arglist = append(arglist, saltCall)
	}

	if !useMaster {
		arglist = append(arglist, local)
	}

	if !colour {
		arglist = append(arglist, noColour)
	}

	if full {
		arglist = append(arglist, fullOutput...)
	} else {
		arglist = append(arglist, changesOnly...)
	}

	if quiet {
		arglist = append(arglist, logQuiet...)
	}

	if outPath != "" {
		arglist = append(arglist, outFile)
		arglist = append(arglist, outPath)
	}

	switch action {
	case "sls":
		if argc == 0 {
			usage(os.Stderr)
			os.Exit(1)
		}

		arglist = append(arglist, "state.sls")
		arglist = append(arglist, argv...)
	case "up", "highstate":
		arglist = append(arglist, "state.highstate")
		arglist = append(arglist, argv...)
	case "sync":
		sync(arglist, argv)
	case "clear":
		arglist = append(arglist, "saltutil.clear_cache")
	default:
		usage(os.Stdout)
		return
	}

	cmd, err := buildCommand(arglist)
	if err != nil {
		fatalf(err, "failed to find %s (is salt installed?)", arglist[0])
	}

	err = cmd.Run()
	if err != nil {
		fatalf(err, "")
	}
}

func init() {
	flag.Usage = func() { usage(os.Stdout) }
}
