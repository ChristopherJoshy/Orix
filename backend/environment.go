package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var environmentKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Existing process values win. Env files never execute or expand shell code.
func loadEnvironment() error {
	path := os.Getenv("ENV_FILE")
	explicit := path != ""
	if !explicit {
		path = ".env"
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return nil
	}
	if err != nil {
		return errors.New("unable to open the selected environment file")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	values := map[string]string{}
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		text = strings.TrimPrefix(text, "export ")
		key, value, ok := strings.Cut(text, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || !environmentKey.MatchString(key) {
			return fmt.Errorf("invalid environment assignment at line %d", line)
		}
		if _, duplicate := values[key]; duplicate {
			return fmt.Errorf("duplicate environment key at line %d", line)
		}
		if strings.HasPrefix(value, "\"") {
			decoded, decodeErr := strconv.Unquote(value)
			if decodeErr != nil {
				return fmt.Errorf("invalid environment quoting at line %d", line)
			}
			value = decoded
		} else if strings.HasPrefix(value, "'") {
			if len(value) < 2 || !strings.HasSuffix(value, "'") {
				return fmt.Errorf("invalid environment quoting at line %d", line)
			}
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	if scanner.Err() != nil {
		return errors.New("unable to parse the environment file")
	}
	for key, value := range values {
		if _, exists := os.LookupEnv(key); !exists {
			if err = os.Setenv(key, value); err != nil {
				return errors.New("unable to apply environment setting")
			}
		}
	}
	return nil
}
