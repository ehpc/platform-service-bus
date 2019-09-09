package adapter

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	rulePkg "platform-service-bus/internal/pkg/rule"
	"strings"
)

// Adapter описывает адаптер для соединения двух сервисов между собой
type Adapter struct {
	Name  string
	Port  int16
	Rules []rulePkg.Rule
}

// Endpoint описывает сгруппированый по пути набор правил
type Endpoint struct {
	path  string
	Rules []rulePkg.Rule
}

// endpointHandler обрабатывает запросы от клиентов
func (endpoint *Endpoint) endpointHandler(w http.ResponseWriter, req *http.Request) {
	for i, rule := range endpoint.Rules {
		if i == len(endpoint.Rules)-1 {
			headers, body := rulePkg.HandleRule(rule, req)
			// Если запрос никуда не уходит, то просто отдаём новый запрос в качестве ответа
			if rule.To.URL == "" {
				responseHeaders := w.Header()
				for _, header := range headers {
					parts := strings.SplitN(header, ":", 2)
					responseHeaders.Set(parts[0], strings.TrimSpace(parts[1]))
				}
				w.Write(body)
			} else { // Если запрос перенаправляется на другой URL
				request, err := http.NewRequest(rule.To.HTTPMethod, rule.To.URL, bytes.NewReader(body))
				if err != nil {
					w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
					return
				}
				requestQuery := request.URL.Query()
				// Прокидываем GET-параметры
				for name, values := range req.URL.Query() {
					for _, value := range values {
						requestQuery.Add(name, value)
					}
				}
				request.URL.RawQuery = requestQuery.Encode()
				// Устанавливаем хедеры
				for _, header := range headers {
					parts := strings.SplitN(header, ":", 2)
					request.Header.Set(parts[0], strings.TrimSpace(parts[1]))
				}
				// Выполняем запрос
				client := &http.Client{}
				response, err := client.Do(request)
				if err != nil {
					w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
					return
				}
				body, err := ioutil.ReadAll(response.Body)
				defer response.Body.Close()
				if err != nil {
					w.Write([]byte(fmt.Sprintf(`{"error": "%v"}`, err)))
					return
				}
				// Прокидываем хедеры из ответа
				responseHeaders := w.Header()
				for name, values := range response.Header {
					for _, value := range values {
						responseHeaders.Set(name, value)
					}
				}
				w.Write(body)
			}
		} else {
			rulePkg.HandleRule(rule, req)
		}
	}
}

// getEndpoints возвращает хэш-таблицу уникальных входящих путей к правилам адаптера
// {"/test" => Rule, ...}
func (adapter *Adapter) getEndpoints() map[string]*Endpoint {
	endpoints := make(map[string]*Endpoint)
	for _, rule := range adapter.Rules {
		if _, prs := endpoints[rule.From.Path]; !prs {
			endpoints[rule.From.Path] = &Endpoint{
				path: rule.From.Path,
			}
		}
		endpoints[rule.From.Path].Rules = append(endpoints[rule.From.Path].Rules, rule)
	}
	return endpoints
}

// getHandler создаёт мультиплексор входящих запросов
func (adapter *Adapter) getHandler() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health-check", HealthCheckHandler)
	// Для каждого URI свой обработчик
	for path, endpoint := range adapter.getEndpoints() {
		mux.HandleFunc(path, endpoint.endpointHandler)
	}
	return mux
}

// StartServer запускает сервер
func (adapter *Adapter) StartServer() {
	mux := adapter.getHandler()
	fmt.Println("Запускаем сервер", adapter)
	http.ListenAndServe(fmt.Sprintf(":%d", adapter.Port), mux)
}

// HealthCheckHandler - обработчик запроса health check
func HealthCheckHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`{"alive": true}`))
}
