package main

import (
	"fmt"
	flag "github.com/dotcloud/docker/pkg/mflag"
	"log"
	"os"
	"strings"
	"text/template"
)

var versionTemplate = template.Must(ParseAsset("version", "templates/version.tmpl"))
var mainTemplate = template.Must(ParseAsset("main", "templates/main.tmpl"))
var commandsTemplate = template.Must(ParseAsset("main", "templates/commands.tmpl"))
var readmeTemplate = template.Must(ParseAsset("readme", "templates/README.tmpl"))
var changelogTemplate = template.Must(ParseAsset("changelog", "templates/CHANGELOG.tmpl"))

var versionGo = Source{
	Name:     "version.go",
	Template: *versionTemplate,
}

var commandsGo = Source{
	Name:     "commands.go",
	Template: *commandsTemplate,
}

var readmeMd = Source{
	Name:     "README.md",
	Template: *readmeTemplate,
}

var changelogMd = Source{
	Name:     "CHANGELOG.md",
	Template: *changelogTemplate,
}

type Application struct {
	Name, Author, Email, Username string
	HasSubCommand                 bool
	SubCommands                   []SubCommand
}

type SubCommand struct {
	Name, DefineName, FunctionName string
}

func ParseAsset(name string, path string) (*template.Template, error) {
	src, err := Asset(path)
	if err != nil {
		return nil, err
	}

	return template.New(name).Parse(string(src))
}

func defineApplication(appName string, inputSubCommands []string, username string) Application {

	hasSubCommand := false
	if inputSubCommands[0] != "" {
		hasSubCommand = true
	}

	gitUsername := GitConfig("user.name")

	if username == "" {
		username = gitUsername
	}

	return Application{
		Name:          appName,
		Author:        gitUsername,
		Email:         GitConfig("user.email"),
		Username:      username,
		HasSubCommand: hasSubCommand,
		SubCommands:   defineSubCommands(inputSubCommands),
	}
}

func defineSubCommands(inputSubCommands []string) []SubCommand {

	var subCommands []SubCommand

	if inputSubCommands[0] == "" {
		return subCommands
	}

	for _, name := range inputSubCommands {
		subCommand := SubCommand{
			Name:         name,
			DefineName:   "command" + ToUpperFirst(name),
			FunctionName: "do" + ToUpperFirst(name),
		}
		subCommands = append(subCommands, subCommand)
	}

	return subCommands
}

func ToUpperFirst(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}

func debug(v ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Println(v...)
	}
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func showVersion() {
	fmt.Fprintf(os.Stderr, "cli-init v%s\n", Version)
}

func showHelp() {
	fmt.Fprintf(os.Stderr, helpText)
}

func main() {

	var (
		flVersion     = flag.Bool([]string{"v", "-version"}, false, "Print version information and quit")
		flHelp        = flag.Bool([]string{"h", "-help"}, false, "Print this message and quit")
		flDebug       = flag.Bool([]string{"-debug"}, false, "Run as DEBUG mode")
		flSubCommands = flag.String([]string{"s", "-subcommands"}, "", "Conma-seplated list of sub-commands to build")
		flForce       = flag.Bool([]string{"f", "-force"}, false, "Overwrite application without prompting")
		flUsername    = flag.String([]string{"u", "-username"}, "", "GitHub username")
	)

	flag.Parse()

	if *flHelp {
		showHelp()
		os.Exit(0)
	}

	if *flVersion {
		showVersion()
		os.Exit(0)
	}

	if *flDebug {
		os.Setenv("DEBUG", "1")
		debug("Run as DEBUG mode")
	}

	inputSubCommands := strings.Split(*flSubCommands, ",")
	debug("inputSubCommands:", inputSubCommands)

	appName := flag.Arg(0)
	debug("appName:", appName)

	if appName == "" {
		fmt.Fprintf(os.Stderr, "Application name must not be blank\n")
		os.Exit(1)
	}

	if _, err := os.Stat(appName); err == nil && *flForce {
		err = os.RemoveAll(appName)
		assert(err)
	}

	if _, err := os.Stat(appName); err == nil {
		fmt.Fprintf(os.Stderr, "%s is already exists, overwrite it? [Y/n]: ", appName)
		var ans string
		_, err := fmt.Scanf("%s", &ans)
		assert(err)

		if ans == "Y" {
			err = os.RemoveAll(appName)
			assert(err)
		} else {
			os.Exit(0)
		}
	}

	// Create directory
	err := os.Mkdir(appName, 0766)
	assert(err)

	application := defineApplication(appName, inputSubCommands, *flUsername)

	// Create README.md
	err = readmeMd.generate(appName, application)
	assert(err)

	// Create CHANGELOG.md
	err = changelogMd.generate(appName, application)
	assert(err)

	// Create verion.go
	err = versionGo.generate(appName, application)
	assert(err)

	// Create <appName>.go
	mainGo := Source{
		Name:     appName + ".go",
		Template: *mainTemplate,
	}
	mainGo.generate(appName, application)
	assert(err)

	// Create commands.go
	if application.HasSubCommand {
		commandsGo.generate(appName, application)
	}

	err = GoFmt(appName)
	assert(err)

	os.Exit(0)
}

const helpText = `Usage: cli-init [options] [application]

cli-init is the easy way to start building command-line app.

Options:

  -s="", --subcommands=""    Comma-separated list of sub-commands to build
  -u="", --username=""       GitHub username
  -f, --force                Overwrite application without prompting 
  -h, --help                 Print this message and quit
  -v, --version              Print version information and quit
  --debug=false              Run as DEBUG mode

Example:

  $ cli-init todo
  $ cli-init -s add,list,delete todo
`
