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
)

const getSafariProcessesCommand = "ps axm -o pid,rss,command | grep com.apple.WebKit.WebContent"
const safariId = "com.apple.WebKit.WebContent"
const maxAllowedUsageMB = 8 * 1024 // 8GB

type safariProcess struct {
	pid   string
	rssMB int
}

type safariProcesses []safariProcess

func (sp safariProcesses) Len() int {
	return len(sp)
}

func (sp safariProcesses) Swap(i, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}

func (sp safariProcesses) Less(i, j int) bool {
	return sp[i].rssMB < sp[j].rssMB
}

func findSafaries() (safariProcesses, error) {

	cmd := "/bin/bash"
	args := []string{"-c", getSafariProcessesCommand}
	rawOutput, err := exec.Command(cmd, args...).Output()

	if err != nil {
		return nil, fmt.Errorf("Can't run \"%v\": %v", getSafariProcessesCommand, err)
	}

	output := string(rawOutput)
	lines := strings.Split(output, "\n")
	var processes safariProcesses

	for _, line := range lines {
		comps := strings.Split(line, " ")
		filteredComps := make([]string, 0)

		for _, comp := range comps {
			trimmedComp := strings.TrimSpace(comp)
			if len(trimmedComp) != 0 {
				filteredComps = append(filteredComps, trimmedComp)
			}
		}

		comps = filteredComps

		if len(comps) < 3 {
			continue
		}

		if !strings.HasSuffix(comps[2], safariId) {
			continue
		}

		var process safariProcess

		process.pid = comps[0]
		rssKB, err := strconv.Atoi(comps[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warn: can't parse \"%v\"\n", line)
			continue
		}

		process.rssMB = rssKB / 1024

		processes = append(processes, process)
	}

	return processes, nil
}

func killProcess(pid string) (output string, err error) {
	cmd := "kill"
	args := []string{"-15", pid}
	rawOutput, err := exec.Command(cmd, args...).Output()

	if err != nil {
		return "", fmt.Errorf("Can't run \"%v\": %v", getSafariProcessesCommand, err)
	}

	output = string(rawOutput)

	return output, nil
}

// Flags
var helpFlagPtr = flag.Bool("h", false, "Show help")
var listFlagPtr = flag.Bool("l", false, "Only list Safari processes")

func main() {

	flag.Parse()

	if *helpFlagPtr {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		return
	}

	safariProcesses, err := findSafaries()

	if err != nil {
		log.Fatalln(err)
	}

	totalUsageMB := 0

	if *listFlagPtr {
		sort.Sort(safariProcesses)

		for i := len(safariProcesses) - 1; i >= 0; i-- {
			process := safariProcesses[i]
			fmt.Printf("Safari process %v uses %dMB.\n", process.pid, process.rssMB)
			totalUsageMB += process.rssMB
		}

		fmt.Printf("Total Safari memory usage is %dMB.\n", totalUsageMB)
		return
	}

	for _, process := range safariProcesses {
		totalUsageMB += process.rssMB
	}

	if totalUsageMB <= maxAllowedUsageMB {
		return
	}

	sort.Sort(safariProcesses)

	for i := len(safariProcesses) - 1; i >= 0; i-- {
		if totalUsageMB <= maxAllowedUsageMB {
			break
		}

		process := safariProcesses[i]

		fmt.Printf("Killing Safari process %v (uses %dMB)...\n", process.pid, process.rssMB)

		output, err := killProcess(process.pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: can't kill process %v: %v\n", process.pid, err)
			continue
		}
		if len(output) != 0 {
			fmt.Println(output)
		}
	}
}
