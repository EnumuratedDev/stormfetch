package main

import (
	"fmt"
	yaml "gopkg.in/yaml.v3"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var configPath = ""
var fetchScriptPath = ""

var config = StormfetchConfig{
	Ascii:             "auto",
	FetchScript:       "auto",
	AnsiiColors:       make([]int, 0),
	ForceConfigAnsii:  false,
	DependencyWarning: true,
}

type StormfetchConfig struct {
	Ascii             string `yaml:"distro_ascii"`
	FetchScript       string `yaml:"fetch_script"`
	AnsiiColors       []int  `yaml:"ansii_colors"`
	ForceConfigAnsii  bool   `yaml:"force_config_ansii"`
	DependencyWarning bool   `yaml:"dependency_warning"`
}

func main() {
	readConfig()
}

func readConfig() {
	// Get home directory
	configDir, _ := os.UserConfigDir()
	// Find valid config directory
	if _, err := os.Stat(path.Join(configDir, "stormfetch/config.yaml")); err == nil {
		configPath = path.Join(configDir, "stormfetch/config.yaml")
	} else if _, err := os.Stat("/etc/stormfetch/config.yaml"); err == nil {
		configPath = "/etc/stormfetch/config.yaml"
	} else {
		log.Fatalf("Config file not found: %s", err.Error())
	}
	// Parse config
	bytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		log.Fatal(err)
	}
	if config.FetchScript == "" {
		log.Fatalf("Fetch script path is empty")
	} else if config.FetchScript != "auto" {
		stat, err := os.Stat(config.FetchScript)
		if err != nil {
			log.Fatalf("Fetch script file not found: %s", err.Error())
		} else if stat.IsDir() {
			log.Fatalf("Fetch script path points to a directory")
		}
	}
	if _, err := os.Stat(path.Join(configDir, "stormfetch/fetch_script.sh")); err == nil {
		fetchScriptPath = path.Join(configDir, "stormfetch/fetch_script.sh")
	} else if _, err := os.Stat("/etc/stormfetch/fetch_script.sh"); err == nil {
		fetchScriptPath = "/etc/stormfetch/fetch_script.sh"
	} else {
		log.Fatalf("Fetch script file not found: %s", err.Error())
	}
	// Show Dependency warning if enabled
	if config.DependencyWarning {
		var dependencies []string
		var missing []string
		for _, depend := range dependencies {
			if _, err := os.Stat(path.Join("/usr/bin/", depend)); err != nil {
				missing = append(missing, depend)
			}
		}
		if len(missing) != 0 {
			fmt.Println("[WARNING] Stormfetch functionality may be limited due to the following dependencies not being installed:")
			for _, depend := range missing {
				fmt.Println(depend)
			}
			fmt.Println("You can disable this warning through your stormfetch config")
		}
	}
	// Fetch ascii art and apply colors
	colorMap := make(map[string]string)
	colorMap["C0"] = "\033[0m"
	setColorMap := func() {
		for i := 0; i < 6; i++ {
			if i > len(config.AnsiiColors)-1 {
				colorMap["C"+strconv.Itoa(i+1)] = "\033[0m"
				continue
			}
			colorMap["C"+strconv.Itoa(i+1)] = fmt.Sprintf("\033[1m\033[38;5;%dm", config.AnsiiColors[i])
		}
	}
	setColorMap()
	ascii := getDistroAsciiArt()
	if strings.HasPrefix(ascii, "#/") {
		firstLine := strings.Split(ascii, "\n")[0]
		if !config.ForceConfigAnsii {
			ansiiColors := strings.Split(strings.TrimPrefix(firstLine, "#/"), ";")
			for i, color := range ansiiColors {
				atoi, err := strconv.Atoi(color)
				if err != nil {
					log.Fatal(err)
				}
				if i < len(config.AnsiiColors) {
					config.AnsiiColors[i] = atoi
				} else {
					config.AnsiiColors = append(config.AnsiiColors, atoi)
				}
			}
			setColorMap()
		}
		ascii = os.Expand(ascii, func(s string) string {
			return colorMap[s]
		})
		ascii = strings.TrimPrefix(ascii, firstLine+"\n")
	} else {
		ascii = os.Expand(ascii, func(s string) string {
			return colorMap[s]
		})
	}
	asciiSplit := strings.Split(ascii, "\n")
	asciiNoColor := StripAnsii(ascii)
	//Execute fetch script
	cmd := exec.Command("/bin/bash", fetchScriptPath)
	cmd.Dir = path.Dir(fetchScriptPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, SetupFetchEnv()...)
	cmd.Env = append(cmd.Env, "C0=\033[0m")
	for key, value := range colorMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	// Print Distro Information
	maxWidth := 0
	for _, line := range strings.Split(asciiNoColor, "\n") {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	final := ""
	for lineIndex, line := range asciiSplit {
		lineVisibleLength := len(strings.Split(asciiNoColor, "\n")[lineIndex])
		lastAsciiColor := ""
		if lineIndex != 0 {
			r := regexp.MustCompile("\033[38;5;[0-9]+m")
			matches := r.FindAllString(asciiSplit[lineIndex-1], -1)
			if len(matches) != 0 {
				lastAsciiColor = r.FindAllString(asciiSplit[lineIndex-1], -1)[len(matches)-1]
			}
		}
		for i := lineVisibleLength; i < maxWidth+5; i++ {
			line = line + " "
		}
		asciiSplit[lineIndex] = lastAsciiColor + line
		str := string(out)
		if lineIndex < len(strings.Split(str, "\n")) {
			line = line + colorMap["C0"] + strings.Split(str, "\n")[lineIndex]
		}
		final += lastAsciiColor + line + "\n"
	}
	fmt.Println(final)
}

func SetupFetchEnv() []string {
	var env = make(map[string]string)
	env["DISTRO_LONG_NAME"] = getDistroInfo().LongName
	env["DISTRO_SHORT_NAME"] = getDistroInfo().ShortName
	env["CPU_MODEL"] = getCPUName()
	env["CPU_THREADS"] = strconv.Itoa(getCPUThreads())
	memory := GetMemoryInfo()
	if memory != nil {
		env["MEM_TOTAL"] = strconv.Itoa(memory.MemTotal)
		env["MEM_USED"] = strconv.Itoa(memory.MemTotal - memory.MemAvailable)
		env["MEM_FREE"] = strconv.Itoa(memory.MemAvailable)
	}
	partitions := getMountedPartitions()
	if len(partitions) != 0 {
		env["MOUNTED_PARTITIONS"] = strconv.Itoa(len(partitions))
		for i, part := range partitions {
			env["PARTITION"+strconv.Itoa(i+1)+"_DEVICE"] = part.Device
			env["PARTITION"+strconv.Itoa(i+1)+"_MOUNTPOINT"] = part.MountPoint
			if part.Label != "" {
				env["PARTITION"+strconv.Itoa(i+1)+"_LABEL"] = part.Label
			}
			env["PARTITION"+strconv.Itoa(i+1)+"_TOTAL_SIZE"] = FormatBytes(part.TotalSize)
			env["PARTITION"+strconv.Itoa(i+1)+"_USED_SIZE"] = FormatBytes(part.UsedSize)
			env["PARTITION"+strconv.Itoa(i+1)+"_FREE_SIZE"] = FormatBytes(part.FreeSize)
		}
	}
	env["DE_WM"] = GetDEWM()
	env["USER_SHELL"] = GetShell()
	env["DISPLAY_PROTOCOL"] = GetDisplayProtocol()
	monitors := getMonitorResolution()
	if len(monitors) != 0 {
		env["CONNECTED_MONITORS"] = strconv.Itoa(len(monitors))
		for i, monitor := range monitors {
			env["MONITOR"+strconv.Itoa(i+1)] = monitor
		}
	}
	if getGPUName() != "" {
		env["GPU_MODEL"] = getGPUName()
	}

	var ret = make([]string, len(env))
	i := 0
	for key, value := range env {
		ret[i] = fmt.Sprintf("%s=%s", key, value)
		i++
	}
	return ret
}
