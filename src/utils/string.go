package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
)

func ZhiYinLouDepartmentName(fullName, separator string, index int) string {
	result := strings.Split(fullName, separator)
	if index > len(result) {
		return ""
	}
	return result[index-1]
}

func TruncateString(s string, maxLength int) string {
	if utf8.RuneCountInString(s) <= maxLength {
		return s
	}

	truncated := []rune(s)[:maxLength]
	return string(truncated) + "..."
}

// RemoveHTMLTags 正则移除HTML标签
func RemoveHTMLTags(input string) string {
	// 定义正则表达式模式，匹配HTML标签
	re := regexp.MustCompile(`<[^>]*>`)

	// 使用正则表达式替换所有HTML标签为空字符串
	output := re.ReplaceAllString(input, "")

	// 替换HTML实体 &nbsp; 为空格
	output = regexp.MustCompile(`&nbsp;`).ReplaceAllString(output, " ")

	return output
}

// Cookie 结构体定义
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// JSONToCurlCookie 将 JSON 格式的 cookie 转换为 curl -b 格式的字符串
func JSONToCurlCookie(jsonStr string) (string, error) {
	var cookies []Cookie
	err := json.Unmarshal([]byte(jsonStr), &cookies)
	if err != nil {
		return "", err
	}

	var cookieStr string
	for i, cookie := range cookies {
		if i > 0 {
			cookieStr += "; "
		}
		cookieStr += fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
	}

	return cookieStr, nil
}

func IsValidJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func StripHTMLTags(html string) string {
	p := bluemonday.NewPolicy()
	p.AllowElements() // 允许所有元素(实际会全部过滤掉)
	str := p.Sanitize(html)
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\\n", "")
	re := regexp.MustCompile(`[\n|\\n]+`)
	str = re.ReplaceAllString(str, "")

	return str
}

func SanitizeForURL(s string) string {
	specialChars := []string{"#", "@", "$", "&", "?", "=", "+", "%", " ", "\\", "/"}
	for _, char := range specialChars {
		s = strings.ReplaceAll(s, char, "*")
	}
	return s
}

// IsAllDigits 检查字符串是否只包含数字
func IsAllDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return len(s) > 0
}

// CleanLLMJSON 清理大模型输出的 JSON 字符串，移除 Markdown 代码块标记
func CleanLLMJSON(input string) string {
	input = strings.TrimSpace(input)
	// 匹配 ```json ... ``` 或 ``` ... ```
	re := regexp.MustCompile("(?s)^```(?:json)?\\n?(.*?)\\n?```$")
	match := re.FindStringSubmatch(input)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return input
}

// CleanURL 清理 URL 字符串中的非法字符（如反引号、空格）
func CleanURL(url string) string {
	return strings.Trim(url, " `\n\r\t")
}

func JsonString(data any) string {
	ret, _ := json.Marshal(data)
	return string(ret)
}

// 从JSON数组字符串中提取特定字段的值、
func ExtractValuesFromJsonArray(jsonArray string) string {
	arr := make([]string, 0)
	if err := json.Unmarshal([]byte(jsonArray), &arr); err != nil {
		fmt.Printf(">>> json.Unmarshal failed. err: %+v \n", err)
		return ""
	}
	if len(arr) == 0 {
		return jsonArray
	}
	return arr[0]
}

// SplitByCharCount 按字符数分割字符串，确保不截断汉字
func SplitByCharCount(s string, maxChars int) []string {
	if maxChars <= 0 {
		return []string{s}
	}

	var result []string
	runes := []rune(s)
	totalRunes := len(runes)

	for i := 0; i < totalRunes; i += maxChars {
		end := i + maxChars
		if end > totalRunes {
			end = totalRunes
		}
		result = append(result, string(runes[i:end]))
	}

	return result
}

// SmartSplit 智能分割，优先在标点或空格处断开
func SmartSplit(s string, maxChars int) []string {
	if maxChars <= 0 {
		return []string{s}
	}

	var result []string
	runes := []rune(s)
	totalRunes := len(runes)

	for i := 0; i < totalRunes; {
		// 确定当前块的结束位置
		end := i + maxChars
		if end > totalRunes {
			end = totalRunes
		}

		// 如果end不在字符串末尾，尝试寻找更好的分割点
		if end < totalRunes {
			// 向前找标点、空格或中文字符边界
			betterEnd := end
			for betterEnd > i && !isGoodBreakPoint(runes[betterEnd-1]) {
				betterEnd--
			}

			// 如果找到了好的分割点，使用它
			if betterEnd > i && betterEnd != end {
				end = betterEnd
			} else {
				// 向后找标点或空格
				for end < totalRunes && !isGoodBreakPoint(runes[end]) {
					end++
				}
			}
		}

		result = append(result, string(runes[i:end]))
		i = end
	}

	return result
}

// isGoodBreakPoint 判断是否是一个好的分割点
func isGoodBreakPoint(r rune) bool {
	// 标点符号、空格、换行等是好的分割点
	return unicode.IsPunct(r) || unicode.IsSpace(r) || r == '、' || r == '，' || r == '。'
}

// SplitLettersAndNumbers 分割字符串，将字母和数字分开
func SplitLettersAndNumbers(s string) (letters, numbers string) {
	var lb, nb strings.Builder
	for _, ch := range s {
		if unicode.IsLetter(ch) {
			lb.WriteRune(ch)
		} else if unicode.IsDigit(ch) {
			nb.WriteRune(ch)
		}
	}
	return lb.String(), nb.String()
}

// SplitByNth 按第n个分隔符拆分字符串
func SplitByNth(text, sep string, n int) (string, string, bool) {
	if n <= 0 {
		return "", "", false
	}

	count := 0
	for i := 0; i < len(text); i++ {
		if strings.HasPrefix(text[i:], sep) {
			count++
			if count == n {
				part1 := text[:i]
				part2 := text[i+len(sep):]
				return part1, part2, true
			}
			i += len(sep) - 1 // 跳过分隔符长度
		}
	}

	return text, "", false
}
