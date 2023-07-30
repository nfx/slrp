package ini

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config map[string]map[string]string

func ParseINI(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := make(Config)
	var currentSection string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || line[0] == ';' || line[0] == '#' {
			continue // Skip empty lines and comments
		}

		if line[0] == '[' && line[len(line)-1] == ']' {
			currentSection = line[1 : len(line)-1]
			cfg[currentSection] = make(map[string]string)
		} else if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if currentSection != "" {
				cfg[currentSection][key] = value
			}
		} else {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cfg, nil
}
