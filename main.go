package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type proc struct {
	user     string
	pid      int
	ppid     int
	pgid     int
	command  string
	children []int
}

type sample struct {
	at    time.Time
	procs map[int]proc
}

func main() {
	command := flag.String("command", "", "Command to run")
	samplingInterval := flag.Int("samplingInterval", 100, "Number of milliseconds to sleep between samples")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "pstree_prof: Sampling every %d ms. Hit Ctrl+C to write to stop\n", *samplingInterval)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case <-c:
			os.Exit(0)
		}
	}()

	if *command == "" {
		fmt.Fprintln(os.Stderr, "pstree_prof: A command must be specified")
		flag.Usage()
		os.Exit(1)
	}

	commandParts := strings.Split(*command, " ")
	cmd, err := startCommandInBackground(commandParts[0], commandParts[1:]...)
	if err != nil {
		panic(err)
	}
	go func() {
		cmd.Wait()
		os.Exit(0)
	}()

	var lastSample sample
	for {
		lastSample = doSample("", lastSample)
		time.Sleep(time.Duration(*samplingInterval) * time.Millisecond)
	}
}

func startCommandInBackground(name string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %s", err)
	}
	return cmd, nil
}

func doSample(pattern string, lastSample sample) sample {
	cols := []string{"user", "pid", "ppid", "pgid", "command"}
	args := []string{"ps", "-axwwo", strings.Join(cols, ",")}
	psCmd := exec.Command(args[0], args[1:]...)
	psOut, err := psCmd.Output()
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(psOut), "\n")
	if len(lines) == 0 {
		panic("pstree_prof: expected at least one line of output")
	}
	// skip header
	lines = lines[1:]
	// if last line is empty, skip
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	procs := make(map[int]proc)
	pids := make([]int, len(lines))
	var pidsOfInterest []int
	for _, line := range lines {
		var colStart, col int
		prevWasSpace := false
		parsedCols := make([]string, len(cols))
		for i, c := range line {
			if col == len(cols)-1 {
				// final column, don't need to search for the end
				// abc___def___ghi
				//    	       ^
				parsedCols[col] = line[i:]
				break
			}

			if !prevWasSpace && c == ' ' {
				// first space char after a string of non-spaces, i.e. the start of the column padding
				// abc___def___ghi
				//    ^
				parsedCols[col] = line[colStart:i]
				col += 1
				prevWasSpace = true
			} else if prevWasSpace && c != ' ' {
				// first non-space after a string of spaces, i.e. the start of a new column
				// abc___def___ghi
				//       ^
				colStart = i
				prevWasSpace = false
			}
		}

		proc := proc{
			user:    parsedCols[0],
			pid:     strictAtoi(parsedCols[1]),
			ppid:    strictAtoi(parsedCols[2]),
			pgid:    strictAtoi(parsedCols[3]),
			command: parsedCols[4],
		}

		if proc.pid == psCmd.Process.Pid {
			// not interested in the `ps ...` command that we started
			continue
		}

		procs[proc.pid] = proc
		pids = append(pids, proc.pid)

		if strings.Contains(proc.command, pattern) {
			pidsOfInterest = append(pidsOfInterest, proc.pid)
		}
	}

	for pid, proc := range procs {
		if parent, ok := procs[proc.ppid]; ok {
			parent.children = append(parent.children, pid)
			procs[proc.ppid] = parent
		}
	}

	type pidToVisit struct {
		pid, depth int
	}
	var pidsToVisit []pidToVisit
	for _, pid := range pidsOfInterest {
		pidsToVisit = append(pidsToVisit, pidToVisit{pid: pid, depth: 0})
	}

	sample := sample{at: time.Now(), procs: make(map[int]proc)}
	for len(pidsToVisit) > 0 {
		pid := pidsToVisit[0]
		pidsToVisit = pidsToVisit[1:]
		if _, ok := sample.procs[pid.pid]; ok {
			continue
		}
		proc := procs[pid.pid]
		sample.procs[pid.pid] = proc

		newPidsToVisit := make([]pidToVisit, len(proc.children))
		for i := 0; i < len(proc.children); i += 1 {
			newPidsToVisit[i] = pidToVisit{pid: proc.children[i], depth: pid.depth + 1}
		}
		// append the new PIDs so they're visited first
		pidsToVisit = append(newPidsToVisit, pidsToVisit...)

		if _, ok := lastSample.procs[pid.pid]; !ok {
			fmt.Printf("STARTED: %d %s\n", pid.pid, proc.command)
		}
	}

	for pid, proc := range lastSample.procs {
		if _, ok := sample.procs[pid]; !ok {
			fmt.Printf("ENDED: %d %s\n", pid, proc.command)
		}
	}

	return sample
}

func strictAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}
