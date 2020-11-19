package secrethub

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func openEditor(path string, secretPaths []string) (map[string]string, error) {
	fpath := os.TempDir() + "secretPaths.txt"
	f, err := os.Create(fpath)
	if err != nil {
		return nil, err
	}

	_, err = f.WriteString(buildFile(path, secretPaths))
	if err != nil {
		return nil, err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "editor"
	}

	cmd := exec.Command(editor, fpath)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	reading, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	err = os.Remove(fpath)
	if err != nil {
		return nil, err
	}

	return buildMap(string(reading)), nil
}

func buildFile(path string, secretPaths []string) string {
	output := "Choose the paths to where your secrets will be written:\n"

	for _, secretPath := range secretPaths {
		output += fmt.Sprintf("%s => %s/%s\n", secretPath,
			path, strings.ToLower(secretPath))
	}
	return output
}

func buildMap(input string) map[string]string {
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Scan()
	locationsMap := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, "=>")
		locationsMap[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
	}
	return locationsMap
}

func getMapKeys(stringMap map[string]string) []string {
	keys := make([]string, 0, len(stringMap))

	for k := range stringMap {
		keys = append(keys, k)
	}
	return keys
}
