package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const getImpudentSafariesCommand = "ps axm -o pid,rss,command | grep com.apple.WebKit.WebContent"
const safariId = "com.apple.WebKit.WebContent"
const killThresholdBytes = 1024 * 1024 // 1GB

type safariProcess struct {
	pid string
	rss int
}

func findSafaries() ([]safariProcess, error) {

	cmd := "/bin/bash"
	args := []string{"-c", getImpudentSafariesCommand}
	rawOutput, err := exec.Command(cmd, args...).Output()

	if err != nil {
		return nil, fmt.Errorf("Can't run \"%v\": %v", getImpudentSafariesCommand, err)
	}

	output := string(rawOutput)
	lines := strings.Split(output, "\n")
	var processes []safariProcess

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
		process.rss, err = strconv.Atoi(comps[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warn: can't parse \"%v\"\n", line)
			continue
		}

		processes = append(processes, process)
	}

	return processes, nil
}

func killProcess(pid string) (output string, err error) {
	cmd := "kill"
	args := []string{"-15", pid}
	rawOutput, err := exec.Command(cmd, args...).Output()

	if err != nil {
		return "", fmt.Errorf("Can't run \"%v\": %v", getImpudentSafariesCommand, err)
	}

	output = string(rawOutput)

	return output, nil
}

func main() {

	safariProcesses, err := findSafaries()

	if err != nil {
		log.Fatalln(err)
	}

	for _, process := range safariProcesses {
		if process.rss >= killThresholdBytes {
			usedMemoryInGb := (float64)(process.rss) / (1024.0 * 1024.0)
			fmt.Printf("Killing process %v (uses %.1fGB)...\n", process.pid, usedMemoryInGb)
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
}
