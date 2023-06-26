package utils

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// 按行使用正则语言查找结构体的属性信息
func ReadInfo(reader *strings.Reader, reg string) (string, error) {
	scanner := bufio.NewScanner(reader)
	var result string
	for {
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		line = strings.TrimSpace(line)
		reg := regexp.MustCompile(reg)
		x := reg.FindAllString(line, -1)
		if x == nil {
			continue
		}
		str := strings.Fields(x[0])
		length := len(str)
		if length < 3 {
			continue
		} else if length == 3 {
			result = str[2]
			return result, nil
		} else {
			i := 3
			result = str[2]
			for {
				if i == length {
					break
				}
				result = result + " " + str[i]
				i += 1

			}
			return result, nil
		}
	}
	return string(""), fmt.Errorf("failed to match struct properties")
}
