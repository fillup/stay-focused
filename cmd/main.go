package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/mitchellh/go-ps"
)

var (
	moduleName          string
	processName         string
	device              string
	runningCheckTimeout int
	refocusTimeout      int
	useV4l2             bool
)

func init() {
	flag.StringVar(&moduleName, "module", "uvcvideo", "The module to check for usage, ex: uvcvideo")
	flag.StringVar(&processName, "proc", "", "The process name to check if running, ex: /opt/zoom/aomhost. If provided this will be used instead of module")
	flag.StringVar(&device, "device", "/dev/video0", "The camera device to use")
	flag.IntVar(&runningCheckTimeout, "check", 1, "How often to check if proc is running in minutes")
	flag.IntVar(&refocusTimeout, "refocus", 10, "How often to refocus camera in seconds while proc is running")
	flag.BoolVar(&useV4l2, "v4l2", false, "Use default v4l2-ctl refocus command. If set argument for refocus command is not required.")
	flag.Parse()
}

func main() {
	cxt, cancelMain := context.WithCancel(context.Background())
	sigchnl := make(chan os.Signal, 1)
	signal.Notify(sigchnl)

	var refocusCommand []string
	if useV4l2 {
		refocusCommand = []string{"v4l2-ctl", "-d", device, "--set-ctrl", "focus_automatic_continuous=1"}
	} else if len(os.Args) > 1 {
		refocusCommand = os.Args[1:]
	}

	if len(refocusCommand) == 0 {
		usage()
		os.Exit(1)
	}

	if processName == "" && moduleName == "" {
		fmt.Println("Error: Either process or module is required")
		usage()
		os.Exit(1)
	}

	recheckInterval := time.Duration(runningCheckTimeout) * time.Minute
	refocusInterval := time.Duration(refocusTimeout) * time.Second

	startedMsg := strings.Builder{}
	startedMsg.WriteString("Stay Focus started at " + time.Now().Format(time.RFC1123Z) + ":\n\tDevice: " + device + "\n")
	if processName != "" {
		startedMsg.WriteString("\tWatching for process: " + processName + "\n")
		startedMsg.WriteString("\tChecking if running every: " + recheckInterval.String() + "\n")
	} else {
		startedMsg.WriteString("\tWatching module for use: " + moduleName + "\n")
		startedMsg.WriteString("\tChecking if in use every: " + recheckInterval.String() + "\n")
	}
	startedMsg.WriteString("\tRefocus command: " + strings.Join(refocusCommand, " ") + "\n")
	startedMsg.WriteString("\tWill run refocus command every: " + refocusInterval.String() + "\n")
	fmt.Println(startedMsg.String())

	ticker := time.NewTicker(recheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if (processName != "" && isProcessRunning(processName)) || (moduleName != "" && isModuleInUse(moduleName)) {
				xcxt, cancelRefocus := context.WithTimeout(cxt, recheckInterval-refocusInterval)
				defer cancelRefocus()
				go handleRefocus(xcxt, refocusCommand, refocusInterval)
			}
		case s := <-sigchnl:
			log.Printf("Received signal: %s, will exit now\n", s.String())
			cancelMain()
			os.Exit(0)
		}
	}
}

func isProcessRunning(proc string) bool {
	procName := strings.ToLower(proc)

	procs, err := ps.Processes()
	if err != nil {
		log.Printf("Error reading process list: %v", err)
		return false
	}

	for _, v := range procs {
		if strings.ToLower(v.Executable()) == procName {
			return true
		}
	}

	return false
}

func isModuleInUse(module string) bool {
	modName := strings.ToLower(module)

	file, err := os.Open("/proc/modules")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Split(scanner.Text(), " ")
		name, used := s[0], s[2]
		if strings.ToLower(name) == modName {
			inUse := used != "0"
			if inUse {
			} else {
			}
			return inUse
		}
	}

	log.Println("Module not found")
	return false
}

func handleRefocus(cxt context.Context, refocusCommand []string, refocusInterval time.Duration) {
	ticker := time.NewTicker(refocusInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cxt.Done():
			return
		case <-ticker.C:
			var cmd *exec.Cmd
			if len(refocusCommand) >= 2 {
				cmd = exec.Command(refocusCommand[0], refocusCommand[1:]...)
			} else {
				cmd = exec.Command(refocusCommand[0])
			}
			if err := cmd.Run(); err != nil {
				log.Printf("Error running refocus command (%s): %s", strings.Join(refocusCommand, " "), err.Error())
			}
		}
	}
}

func usage() {
	fmt.Printf(`
Stay Focused!

Stay Focused monitors for a given process or module to be in use and when it is runs the given comment to 
hopefully tell your camera to refocus video. 

Usage:

	stay-focused -proc {name} -check {minutes} -refocus {seconds} refocus command --with args

Examples:

	stay-focused -proc /opt/zoom/aomhost -check 5 -refocus 30 -v4l2
	stay-focused -module uvcvideo -device /dev/video0 -check 10 -refocus 10 /run/this/command --to --refocus \
			--my camera

Flags:

	proc:		The name of the process to monitor for as would show up when running "ps", 
			example: /opt/zoom/aomhost
	module:		The name of the module to monitor for use instead of process
	device:		The device to refocus if using default v4l2 command but needing different device
	check:		The interval in minutes to check for proc to be running
	refocus:	The interval in seconds to execute refocus command
	v4l2:		If you use v4l2-ctl to control your camera this flag will use the 
				standard/common command to refocus your camera.

Using v4l2-ctl:
	If you enable the v4l2 flag the following command will be used to refocus your camera. 
	If this does not work for you, you'll need to provide you're own command to do so.

		v4l2-ctl -d /dev/video0 --set-ctrl focus_automatic_continuous=1

Arguments:

	After the flags are set (all are optional), provide the command you would run to refocus your 
	camera. 

`)
}
