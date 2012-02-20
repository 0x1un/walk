// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package walk

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

type IniFileSettings struct {
	data map[string]string
}

func NewIniFileSettings() *IniFileSettings {
	return &IniFileSettings{data: make(map[string]string)}
}

func (ifs *IniFileSettings) Get(key string) (string, bool) {
	val, ok := ifs.data[key]
	return val, ok
}

func (ifs *IniFileSettings) Put(key, value string) os.Error {
	if strings.IndexAny(key, "=\r\n") > -1 || strings.IndexAny(value, "\r\n") > -1 {
		return newError("either key or value contains at least one of the invalid characters '=\\r\\n'")
	}

	ifs.data[key] = value

	return nil
}

func (ifs *IniFileSettings) filePath() (string, os.Error) {
	appDataPath, err := AppDataPath()
	if err != nil {
		return "", err
	}

	return path.Join(
		appDataPath,
		appSingleton.OrganizationName(),
		appSingleton.ProductName(),
		"settings.ini"), nil
}

func (ifs *IniFileSettings) fileExists() (bool, os.Error) {
	filePath, err := ifs.filePath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(filePath)
	if err != nil {
		// FIXME: Not necessarily a file does not exist error.
		return false, nil
	}

	return true, nil
}

func (ifs *IniFileSettings) withFile(flags int, f func(file *os.File) os.Error) os.Error {
	filePath, err := ifs.filePath()

	dirPath, _ := path.Split(filePath)
	if err := os.MkdirAll(dirPath, 0644); err != nil {
		return wrapError(err)
	}

	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return wrapError(err)
	}
	defer file.Close()

	return f(file)
}

func (ifs *IniFileSettings) Load() os.Error {
	exists, err := ifs.fileExists()
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	return ifs.withFile(os.O_RDONLY, func(file *os.File) os.Error {
		lineBytes := make([]byte, 0, 4096)
		reader := bufio.NewReader(file)

		for {
			lineBytes = lineBytes[:0]

			for {
				ln, isPrefix, err := reader.ReadLine()
				if err != nil {
					if err == os.EOF {
						return nil
					}
					return wrapError(err)
				}

				lineBytes = append(lineBytes, ln...)

				if !isPrefix {
					break
				}
			}

			lineStr := string(lineBytes)
			assignIndex := strings.Index(lineStr, "=")
			if assignIndex == -1 {
				return newError("bad line format: missing '='")
			}

			key := strings.TrimSpace(lineStr[:assignIndex])
			val := strings.TrimSpace(lineStr[assignIndex+1:])

			ifs.data[key] = val
		}

		return nil
	})
}

func (ifs *IniFileSettings) Save() os.Error {
	return ifs.withFile(os.O_CREATE|os.O_TRUNC|os.O_WRONLY, func(file *os.File) os.Error {
		bufWriter := bufio.NewWriter(file)

		for key, val := range ifs.data {
			line := fmt.Sprintf("%s=%s\n", key, val)

			if _, err := bufWriter.WriteString(line); err != nil {
				return wrapError(err)
			}
		}

		return bufWriter.Flush()
	})
}
