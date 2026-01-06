package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1"

const logo = `
  ╭────────────────────────────────╮
  │  ▀█▀ █▀█ █▄▀ █ █▀▄▀█ █ █ █▄ █  │
  │   █  █▄█ █ █ █ █ ▀ █ █▄█ █ ▀█  │
  ╰────────────────────────────────╯
`

const help = `tokimun, lua without the pet peeves

USAGE:
    tokimun <command> [options] [file]

COMMANDS:
    compile, c    Compile .tkm file(s) to Lua
    run, r        Compile and run with Lua interpreter  
    watch, w      Watch files and recompile on change
    version, v    Print version information
    help, h       Show this help message

OPTIONS:
    -o, --output <file>    Output file (default: input with .lua extension)
    -p, --print            Print compiled output to stdout
    -q, --quiet            Suppress non-error output
    --stdout               Write to stdout instead of file

EXAMPLES:
    tokimun compile main.tkm              # Creates main.lua
    tokimun compile main.tkm -o out.lua   # Creates out.lua
    tokimun compile src/*.tkm             # Compile multiple files
    tokimun run main.tkm                  # Compile and execute
    tokimun c main.tkm -p                 # Print compiled Lua`

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Print(logo)
		fmt.Println("\n  Run 'tokimun help' for usage information.")
		os.Exit(0)
	}

	command := args[0]
	args = args[1:]

	switch command {
	case "compile", "c":
		handleCompile(args)
	case "run", "r":
		handleRun(args)
	case "watch", "w":
		handleWatch(args)
	case "version", "v", "--version", "-v":
		fmt.Printf("tokimun v%s\n", version)
	case "help", "h", "--help", "-h":
		fmt.Print(logo)
		fmt.Println(help)
	default:
		// Assume it's a file to compile
		handleCompile(append([]string{command}, args...))
	}
}

type CompileOptions struct {
	OutputFile string
	PrintOnly  bool
	Quiet      bool
	ToStdout   bool
}

func parseCompileOptions(args []string) ([]string, CompileOptions) {
	opts := CompileOptions{}
	files := []string{}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-o", "--output":
			if i+1 < len(args) {
				opts.OutputFile = args[i+1]
				i += 2
			} else {
				fatal("error: -o requires an output file argument")
			}
		case "-p", "--print":
			opts.PrintOnly = true
			i++
		case "-q", "--quiet":
			opts.Quiet = true
			i++
		case "--stdout":
			opts.ToStdout = true
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				fatal("error: unknown option '%s'", arg)
			}
			files = append(files, arg)
			i++
		}
	}

	return files, opts
}

func handleCompile(args []string) {
	files, opts := parseCompileOptions(args)

	if len(files) == 0 {
		fatal("error: no input files specified\n\nUsage: tokimun compile <file.tkm> [options]")
	}

	// Expand globs
	expandedFiles := []string{}
	for _, pattern := range files {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			fatal("error: invalid file pattern '%s': %v", pattern, err)
		}
		if len(matches) == 0 {
			// Not a glob, treat as literal filename
			expandedFiles = append(expandedFiles, pattern)
		} else {
			expandedFiles = append(expandedFiles, matches...)
		}
	}

	for _, file := range expandedFiles {
		if err := compileFile(file, opts); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func compileFile(inputPath string, opts CompileOptions) error {
	// Validate input file
	if !strings.HasSuffix(inputPath, ".tkm") {
		return fmt.Errorf("'%s' is not a .tkm file", inputPath)
	}

	// Read input
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("cannot read '%s': %v", inputPath, err)
	}

	// Compile
	output, err := Compile(string(source))
	if err != nil {
		return fmt.Errorf("%s: %v", inputPath, err)
	}

	// Handle output
	if opts.PrintOnly || opts.ToStdout {
		fmt.Print(output)
		return nil
	}

	// Determine output path
	outputPath := opts.OutputFile
	if outputPath == "" {
		outputPath = strings.TrimSuffix(inputPath, ".tkm") + ".lua"
	}

	// Write output
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("cannot write '%s': %v", outputPath, err)
	}

	if !opts.Quiet {
		fmt.Printf("✓ %s → %s\n", inputPath, outputPath)
	}

	return nil
}

func handleRun(args []string) {
	files, opts := parseCompileOptions(args)

	if len(files) == 0 {
		fatal("error: no input file specified\n\nUsage: tokimun run <file.tkm>")
	}

	if len(files) > 1 {
		fatal("error: can only run one file at a time")
	}

	inputPath := files[0]

	// Compile to temp file
	source, err := os.ReadFile(inputPath)
	if err != nil {
		fatal("error: cannot read '%s': %v", inputPath, err)
	}

	output, err := Compile(string(source))
	if err != nil {
		fatal("error: %s: %v", inputPath, err)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "tokimun-*.lua")
	if err != nil {
		fatal("error: cannot create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(output); err != nil {
		fatal("error: cannot write temp file: %v", err)
	}
	tmpFile.Close()

	if !opts.Quiet {
		fmt.Printf("✓ compiled %s\n", inputPath)
		fmt.Println("─────────────────────────")
	}

	// Try different Lua interpreters
	interpreters := []string{"lua", "luajit", "lua5.4", "lua5.3", "lua5.2", "lua5.1"}

	var interpreter string
	for _, interp := range interpreters {
		if _, err := execLookPath(interp); err == nil {
			interpreter = interp
			break
		}
	}

	if interpreter == "" {
		fatal("error: no Lua interpreter found. Install lua or luajit.")
	}

	// Execute
	cmd := execCommand(interpreter, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func handleWatch(args []string) {
	files, _ := parseCompileOptions(args)

	if len(files) == 0 {
		fatal("error: no files to watch\n\nUsage: tokimun watch <file.tkm>")
	}

	fmt.Println("Watch mode not yet implemented in v0.1")
	fmt.Println("For now, use a file watcher like entr or watchexec:")
	fmt.Println()
	fmt.Println("  ls *.tkm | entr -c tokimun compile /_")
	fmt.Println("  watchexec -e tkm -- tokimun compile *.tkm")
}

// Compile compiles tokimun source to Lua
func Compile(source string) (string, error) {
	lexer := NewLexer(source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return "", err
	}

	compiler := NewCompiler(tokens)
	return compiler.Compile()
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// Exec helpers (platform independent)
func execLookPath(file string) (string, error) {
	// Simple PATH lookup
	paths := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range paths {
		path := filepath.Join(dir, file)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("not found")
}

type execCmd struct {
	path   string
	args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func execCommand(name string, args ...string) *execCmd {
	path, _ := execLookPath(name)
	return &execCmd{path: path, args: append([]string{name}, args...)}
}

func (c *execCmd) Run() error {
	// Use os/exec for actual execution
	return runCommand(c.path, c.args[1:], c.Stdin, c.Stdout, c.Stderr)
}

// This will be in a separate file for the actual os/exec import
func runCommand(path string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Lazy import to avoid issues
	cmd := &osExecCmd{path: path, args: args, stdin: stdin, stdout: stdout, stderr: stderr}
	return cmd.run()
}

type osExecCmd struct {
	path   string
	args   []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (c *osExecCmd) run() error {
	// Import os/exec inline
	proc, err := os.StartProcess(c.path, append([]string{c.path}, c.args...), &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		return err
	}

	state, err := proc.Wait()
	if err != nil {
		return err
	}

	if !state.Success() {
		return fmt.Errorf("process exited with error")
	}

	return nil
}
