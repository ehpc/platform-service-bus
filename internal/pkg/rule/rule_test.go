package rule

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRule(t *testing.T) {
	table := []struct {
		name     string
		rule     Rule
		request  *http.Request
		expected string
	}{
		{
			name: "Составление нового запроса из GET-запроса",
			rule: Rule{
				From: From{
					Path:       "/test1",
					HTTPMethod: "GET",
				},
				To: To{
					Data: `{"rule": "test1", "query": "%QUERY[q1]%%QUERY[q2]%"}`,
				},
			},
			request:  httptest.NewRequest("GET", "/test1?q1=value1&q2=value2", strings.NewReader("")),
			expected: `{"rule": "test1", "query": "value1value2"}`,
		},
		{
			name: "Составление нового запроса по регулярным выражениям",
			rule: Rule{
				From: From{
					Path:       "/test2",
					HTTPMethod: "GET",
				},
				To: To{
					Data: `{"rule": "test2", "response": "%REGEX[test>([^<]+)][1]%"}`,
				},
			},
			request:  httptest.NewRequest("GET", "/test2?q1=value1&q2=value2", strings.NewReader("<test>somevalue</test>")),
			expected: `{"rule": "test2", "response": "somevalue"}`,
		},
		{
			name: "Подстановка тела запроса в новый запрос",
			rule: Rule{
				From: From{
					Path:       "/test3",
					HTTPMethod: "GET",
				},
				To: To{
					Data: `{"rule": "test3", "response": "%BODY%"}`,
				},
			},
			request:  httptest.NewRequest("GET", "/test3", strings.NewReader("<test>somevalue</test>")),
			expected: `{"rule": "test3", "response": "<test>somevalue</test>"}`,
		},
		{
			name: "Корректная обработка отсутствующих параметров GET-запроса",
			rule: Rule{
				From: From{
					Path:       "/test4",
					HTTPMethod: "GET",
				},
				To: To{
					Data: `{"rule": "test4", "query": "%QUERY[q1]%%QUERY[q2]%"}`,
				},
			},
			request:  httptest.NewRequest("GET", "/test1?q1=value1", strings.NewReader("")),
			expected: `{"rule": "test4", "query": "value1"}`,
		},
	}
	for _, item := range table {
		t.Run(item.name, func(t *testing.T) {
			_, body := HandleRule(item.rule, item.request)
			if string(body) != item.expected {
				t.Errorf("Неверный новый запрос. Expected %v, got %q, len %d", item.expected, body, len(body))
			}
		})
	}
}
