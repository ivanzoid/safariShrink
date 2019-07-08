package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"sort"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

const getSafariProcessesCommand = "ps axm -o pid,rss,command | grep com.apple.WebKit.WebContent"
const safariId = "com.apple.WebKit.WebContent"

var limitMB int64 = 8 * 1024

type safariProcess struct {
	pid   string
	rssMB int64
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
		rssKB, err := strconv.ParseInt(comps[1], 10, 64)
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

func loadConfig(configPathRelativeToHome string) (config map[string]interface{}, err error) {
	user, _ := user.Current()
	filePath := path.Join(user.HomeDir, configPathRelativeToHome)

	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	var rawData interface{}
	err = yaml.Unmarshal(bytes, &rawData)
	if err != nil {
		return
	}

	halfRawData, ok := rawData.(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("Yaml root object is not dictionary")
	}

	config = make(map[string]interface{})

	for key, value := range halfRawData {
		if keyAsString, ok := key.(string); ok {
			config[keyAsString] = value
		} else {
			fmt.Printf("Key '%v' is not string - ignoring.\n", key)
		}
	}

	return config, nil
}

func interfaceToInt64(value interface{}) int64 {
	if value == nil {
		return 0
	}
	switch typedValue := value.(type) {
	case int64:
		return typedValue
	case int8:
		return int64(typedValue)
	case int16:
		return int64(typedValue)
	case int32:
		return int64(typedValue)
	case uint8:
		return int64(typedValue)
	case uint16:
		return int64(typedValue)
	case uint32:
		return int64(typedValue)
	case int:
		return int64(typedValue)
	case uint:
		return int64(typedValue)
	default:
		return 0
	}
}

func configReadInt64(config map[string]interface{}, key string) (int64, bool) {
	if valueRaw, ok := config[key]; ok {
		return interfaceToInt64(valueRaw), true
	}
	return 0, false
}

// Flags
var helpFlagPtr = flag.Bool("h", false, "Show help")
var listFlagPtr = flag.Bool("l", false, "Only list Safari processes")
var forceFlagPtr = flag.Bool("f", false, "Force kill all Safari")

func main() {

	flag.Parse()

	if *helpFlagPtr {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		return
	}

	config, err := loadConfig(".safariShrink/config.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't read config: %v\n", err)
	}
	if limitMBRead, ok := configReadInt64(config, "limitMB"); ok {
		limitMB = limitMBRead
	}

	//fmt.Printf("limitMB = %v\n", limitMB)

	safariProcesses, err := findSafaries()

	if err != nil {
		log.Fatalln(err)
	}

	var totalUsageMB int64 = 0

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

	if !*forceFlagPtr && totalUsageMB <= limitMB {
		return
	}

	fmt.Printf("Total Safari memory usage is %dMB.\n", totalUsageMB)

	sort.Sort(safariProcesses)

	for i := len(safariProcesses) - 1; i >= 0; i-- {
		if !*forceFlagPtr && totalUsageMB <= limitMB {
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

		totalUsageMB -= process.rssMB
	}
}
