package adapter

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	rulePkg "platform-service-bus/internal/pkg/rule"
	"strings"
	"testing"
)

func TestGetEndpoints(t *testing.T) {
	adapter := &Adapter{
		Rules: []rulePkg.Rule{
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test",
					HTTPMethod: "GET",
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test",
					HTTPMethod: "POST",
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test",
					HTTPMethod: "GET",
				},
			},
		},
	}
	endpoints := adapter.getEndpoints()
	if len(endpoints) != 1 {
		t.Errorf("Ожидаем один endpoint, получили %d", len(endpoints))
	}
	if _, prs := endpoints["/test"]; !prs {
		t.Errorf("Ожидаем endpoint /test, получили %v", endpoints)
	}
	if len(endpoints["/test"].Rules) != 3 {
		t.Errorf("Ожидаем 3 правила, получили %d", len(endpoints["/test"].Rules))
	}
}

func TestStartServer(t *testing.T) {
	table := []struct {
		name              string
		url               string
		method            string
		body              string
		expected          string
		expectedHeaders   []string
		expectedSubstring string
		expectedError     bool
	}{
		{
			name:          "Health check",
			url:           "/health-check",
			method:        "GET",
			expected:      `{"alive": true}`,
			expectedError: false,
		},
		{
			name:          "GET query",
			url:           "/test1?q1=1&q2=2",
			method:        "GET",
			expected:      `{"rule": "test1", "query": "12"}`,
			expectedError: false,
		},
		{
			name:          "POST body",
			url:           "/test2?q1=1&q2=2",
			method:        "POST",
			body:          "123",
			expected:      `{"rule": "test1", "query": "12", "body": "123"}`,
			expectedError: false,
		},
		{
			name:          "Неверный запрос",
			url:           "/notfound",
			method:        "GET",
			expected:      "404 page not found\n",
			expectedError: true,
		},
		{
			name:          "Несколько обработчиков одного URI возвращают ответ последнего",
			url:           "/test3?q1=1&q2=2",
			method:        "GET",
			expected:      `{"rule": "test1", "query": "21"}`,
			expectedError: false,
		},
		{
			name:     "Хедеры передаются корректно",
			url:      "/test4",
			method:   "GET",
			expected: `{"rule": "test4"}`,
			expectedHeaders: []string{
				"Content-Type: text/html",
				"Custom-Header: true",
			},
			expectedError: false,
		},
		{
			name:              "Перенаправление GET-запроса",
			url:               "/test5?q1=1&q2=2",
			method:            "GET",
			expectedSubstring: "args\": {\n    \"p\": \"2\", \n    \"q1\": \"1\", \n    \"q2\": \"2\"\n  }",
			expectedError:     false,
		},
		{
			name:              "Перенаправление POST-запроса",
			url:               "/test6",
			method:            "POST",
			expectedSubstring: `"data": "<test>post</test>"`,
			expectedError:     false,
		},
		{
			name:              "Перенаправление GET-запроса на POST-запрос сохраняет GET-параметры",
			url:               "/test7?q1=1&q2=2",
			method:            "GET",
			expectedSubstring: "args\": {\n    \"q1\": \"1\", \n    \"q2\": \"2\"\n  }",
			expectedError:     false,
		},
	}

	adapter := &Adapter{
		Rules: []rulePkg.Rule{
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test1",
					HTTPMethod: "GET",
				},
				To: rulePkg.To{
					Data: `{"rule": "test1", "query": "%QUERY[q1]%%QUERY[q2]%"}`,
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test2",
					HTTPMethod: "POST",
				},
				To: rulePkg.To{
					Data: `{"rule": "test1", "query": "%QUERY[q1]%%QUERY[q2]%", "body": "%BODY%"}`,
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test3",
					HTTPMethod: "GET",
				},
				To: rulePkg.To{
					Data: `{"rule": "test1", "query": "wrong"}`,
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test3",
					HTTPMethod: "GET",
				},
				To: rulePkg.To{
					Data: `{"rule": "test1", "query": "%QUERY[q2]%%QUERY[q1]%"}`,
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test4",
					HTTPMethod: "POST",
				},
				To: rulePkg.To{
					Headers: []string{
						"Content-Type:text/html",
						"Custom-Header: true",
					},
					Data: `{"rule": "test4"}`,
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test5",
					HTTPMethod: "GET",
				},
				To: rulePkg.To{
					URL: "https://httpbin.org/get?p=2",
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test6",
					HTTPMethod: "POST",
				},
				To: rulePkg.To{
					URL:        "https://httpbin.org/post",
					HTTPMethod: "POST",
					Data:       "<test>post</test>",
				},
			},
			rulePkg.Rule{
				From: rulePkg.From{
					Path:       "/test7",
					HTTPMethod: "GET",
				},
				To: rulePkg.To{
					URL:        "https://httpbin.org/post",
					HTTPMethod: "POST",
					Data:       "<test>post</test>",
				},
			},
		},
	}
	mux := adapter.getHandler()

	// Создаём тестовый сервер
	server := httptest.NewServer(mux)
	defer server.Close()
	client := server.Client()

	for _, item := range table {
		t.Run(item.name, func(t *testing.T) {
			var response *http.Response
			var err error
			// Выполняем запрос
			if item.method == "GET" {
				response, err = client.Get(server.URL + item.url)
			} else {
				reader := strings.NewReader(item.body)
				response, err = client.Post(server.URL+item.url, "text/plain; charset=utf-8", reader)
			}
			if err != nil {
				t.Errorf("Ошибка запроса. Expected nil, got %v", err)
			}
			// Проверяем корректность статуса
			status := response.StatusCode
			if !item.expectedError && status != http.StatusOK {
				t.Errorf("Неверный статус. Expected %v, got %v", http.StatusOK, status)
			} else if item.expectedError && status == http.StatusOK {
				t.Errorf("Неверный статус. Expected not %v, got %v", http.StatusOK, status)
			}
			// Проверяем ответ сервера
			body, err := ioutil.ReadAll(response.Body)
			defer response.Body.Close()
			if err != nil {
				t.Errorf("Ошибка чтения ответа. Expected nil, got %v", err)
			}
			if item.expectedSubstring != "" && strings.Index(string(body), item.expectedSubstring) == -1 {
				t.Errorf("Неверный ответ. Expected susbstring %v, got %q, len %d", item.expectedSubstring, body, len(body))
			} else if item.expectedSubstring == "" && string(body) != item.expected {
				t.Errorf("Неверный ответ. Expected %v, got %q, len %d", item.expected, body, len(body))
			}
			// Проверяем хедеры
			for _, expectedHeader := range item.expectedHeaders {
				parts := strings.SplitN(expectedHeader, ":", 2)
				values, prs := response.Header[parts[0]]
				if !prs {
					t.Errorf("Не найден ожидаемый header %v, got %v", parts[0], response.Header)
					continue
				}
				if values[0] != strings.TrimSpace(parts[1]) {
					t.Errorf("Неверный header. Expected %v, got %v", expectedHeader, values[0])
				}
			}
		})
	}
}
