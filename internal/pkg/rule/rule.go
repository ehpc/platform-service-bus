package rule

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// queryRx регулярка для подстановки GET-параметров
var queryRx = regexp.MustCompile(`%QUERY\[([^]]+)\]%`)

// regexpRx регулярка для подстановки результатов поиска по регулярным выражениям
var regexpRx = regexp.MustCompile(`%REGEX\[(.+?)\]\[(\d+)\]%`)

// Rule описывает правило адаптера
type Rule struct {
	From From
	To   To
}

// From описывает входящий запрос сервиса
type From struct {
	Path       string
	HTTPMethod string `json:"http-method"`
}

// To описывает исходящий запрос сервиса
type To struct {
	URL        string
	HTTPMethod string `json:"http-method"`
	Headers    []string
	Data       string
	DataFile   string `json:"data-file"`
}

// filesCache кэш для подгруженных шаблонов
var filesCache = make(map[string][]byte)

// getFileContents подгружает файл и кэширует данные
func getFileContents(fileName string) []byte {
	data, prs := filesCache[fileName]
	if !prs {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			fmt.Println(err)
		} else {
			filesCache[fileName] = data
		}
		return data
	}
	return data
}

// replaceAllStringSubmatchFunc заменяет все вхождения с помощью функции, принимающей submatches
func replaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0
	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			groups = append(groups, str[v[i]:v[i+1]])
		}
		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}
	return result + str[lastIndex:]
}

// HandleRule формирует ответ согласно правилу адаптера
func HandleRule(rule Rule, req *http.Request) ([]string, []byte) {
	var response string
	query := req.URL.Query()
	body, _ := ioutil.ReadAll(req.Body)
	// Достаём шаблон ответа
	var responseTemplate string
	if rule.To.DataFile != "" {
		responseTemplate = string(getFileContents(rule.To.DataFile))
	} else {
		responseTemplate = rule.To.Data
	}
	// Делаем подстановки GET-параметров
	response = replaceAllStringSubmatchFunc(queryRx, responseTemplate, func(groups []string) string {
		if len(query[groups[1]]) == 1 {
			return query[groups[1]][0]
		}
		return ""
	})
	// Делаем подстановки REGEXP
	response = replaceAllStringSubmatchFunc(regexpRx, response, func(groups []string) string {
		searchRx, err := regexp.Compile(groups[1])
		if err != nil {
			fmt.Println("Ошибка компиляции регулярного выражения", err)
			return ""
		}
		submatchIndex, err := strconv.Atoi(groups[2])
		if err != nil {
			fmt.Println("Недопустимый индекс группы регулярного выражения", err)
			return ""
		}
		if len(body) > 0 {
			matches := searchRx.FindSubmatch(body)
			if submatchIndex >= 0 && submatchIndex < len(matches) {
				return string(matches[submatchIndex])
			}
			fmt.Println("Группа регулярного выражения не существует по указанному индексу", err)
			return ""
		}
		return ""
	})
	// Делаем подстановки тела запроса
	response = strings.ReplaceAll(response, "%BODY%", string(body))
	return rule.To.Headers, []byte(response)
}
