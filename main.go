package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type proc struct {
	User     string `json:"user"`
	Pid      int    `json:"pid"`
	Ppid     int    `json:"ppid"`
	Pgid     int    `json:"pgid"`
	Command  string `json:"command"`
	Children []int  `json:"children"`
}

type sample struct {
	At    time.Time    `json:"at"`
	Procs map[int]proc `json:"procs"`
}

func main() {
	command := flag.String("command", "", "Command to run")
	outputMode := flag.String("outputMode", "count", "Method used to summarize the samples")
	samplingInterval := flag.Int("samplingInterval", 100, "Number of milliseconds to sleep between samples")
	flag.Parse()

	if *command == "" {
		flag.Usage()
		log.Fatalln("a non-empty command must be specified")
	}

	log.SetPrefix("pstree_prof: ")
	log.Printf("sampling every %dms\n", *samplingInterval)

	samples := make([]sample, 1)
	commandParts := strings.Split(*command, " ")
	cmd, err := startCommandInBackground(commandParts[0], commandParts[1:], func() {
		switch *outputMode {
		case "count":
			countCommandOccurrences(samples)
		case "json":
		default:
			log.Fatalf("unrecognized outputMode: %s\n", *outputMode)
		}
		os.Exit(0) // why do I need to do this?
	})
	if err != nil {
		log.Fatalln(err)
	}

	var lastSample sample
	for {
		lastSample = sampleProcs(cmd.Process.Pid, lastSample)
		samples = append(samples, lastSample)
		time.Sleep(time.Duration(*samplingInterval) * time.Millisecond)
	}
}

func sampleProcs(pid int, lastSample sample) sample {
	cols := []string{"user", "pid", "ppid", "pgid", "command"}
	args := []string{"ps", "-axwwo", strings.Join(cols, ",")}
	psCmd := exec.Command(args[0], args[1:]...)
	psOut, err := psCmd.Output()
	if err != nil {
		log.Fatalln(fmt.Errorf("could not start `ps`: %s", err))
	}

	lines := strings.Split(string(psOut), "\n")
	if len(lines) == 0 {
		log.Fatalln("expected at least one line of output from `ps`")
	}

	// skip header
	lines = lines[1:]
	// if last line is empty, skip
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	procs := make(map[int]proc)
	for _, line := range lines {
		proc := parseLineAsProc(line, cols)

		if proc.Pid == psCmd.Process.Pid {
			// not interested in the `ps ...` command that we started
			continue
		}

		procs[proc.Pid] = proc
	}

	for pid, proc := range procs {
		if parent, ok := procs[proc.Ppid]; ok {
			parent.Children = append(parent.Children, pid)
			procs[proc.Ppid] = parent
		}
	}

	type pidToVisit struct {
		pid, depth int
	}
	pidsToVisit := []pidToVisit{
		{pid, 0},
	}

	sample := sample{At: time.Now(), Procs: make(map[int]proc)}
	for len(pidsToVisit) > 0 {
		pid := pidsToVisit[0]
		pidsToVisit = pidsToVisit[1:]
		if _, ok := sample.Procs[pid.pid]; ok {
			continue
		}
		proc := procs[pid.pid]
		sample.Procs[pid.pid] = proc

		newPidsToVisit := make([]pidToVisit, len(proc.Children))
		for i := 0; i < len(proc.Children); i += 1 {
			newPidsToVisit[i] = pidToVisit{pid: proc.Children[i], depth: pid.depth + 1}
		}
		// append the new PIDs so they're visited first
		pidsToVisit = append(newPidsToVisit, pidsToVisit...)
	}

	return sample
}

func parseLineAsProc(line string, cols []string) proc {
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

	return proc{
		User:    parsedCols[0],
		Pid:     strictAtoi(parsedCols[1]),
		Ppid:    strictAtoi(parsedCols[2]),
		Pgid:    strictAtoi(parsedCols[3]),
		Command: parsedCols[4],
	}
}

func countCommandOccurrences(samples []sample) {
	counts := make(map[string]int)
	for _, sample := range samples {
		for _, proc := range sample.Procs {
			if count, ok := counts[proc.Command]; ok {
				counts[proc.Command] = count + 1
			} else {
				counts[proc.Command] = 1
			}
		}
	}

	type countAndCommand struct {
		count   int
		command string
	}
	countsAndCommands := make([]countAndCommand, len(counts))
	for command, count := range counts {
		countsAndCommands = append(countsAndCommands, countAndCommand{count, command})
	}

	sort.SliceStable(countsAndCommands, func(i, j int) bool {
		return countsAndCommands[i].count > countsAndCommands[j].count
	})

	fmt.Println("count\tcommand")
	for _, cAndC := range countsAndCommands {
		if cAndC.count == 0 {
			continue
		}
		fmt.Printf("%d\t%s\n", cAndC.count, cAndC.command)
	}
}

func startCommandInBackground(name string, args []string, afterCommand func()) (*exec.Cmd, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println("start of output from command:")
	err := cmd.Start()
	go func() {
		cmd.Wait()
		log.Println("end of output from command")
		afterCommand()
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %s", err)
	}
	return cmd, nil
}

func strictAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}
