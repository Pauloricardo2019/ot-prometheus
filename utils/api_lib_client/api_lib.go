package api_lib_client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type RetryStrategy interface {
	NextWaitTime(retry int) time.Duration
}

type ExponentialRetryStrategy struct {
}

func (ers *ExponentialRetryStrategy) NextWaitTime(retry int) time.Duration {
	return time.Duration(retry+1) * time.Second
}

type APIRetry struct {
	RetryStrategy RetryStrategy // Estratégia de retentativa
}

func NewAPIRetry() *APIRetry {
	return &APIRetry{
		RetryStrategy: &ExponentialRetryStrategy{},
	}
}

func (ar *APIRetry) SetRetryStrategy(strategy RetryStrategy) {
	ar.RetryStrategy = strategy
}

func (ar *APIRetry) getResponseBody(method, url string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Erro ao criar a requisição: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Erro ao realizar a requisição: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Erro ao ler o corpo da resposta: %v", err)
	}

	return responseBody, nil
}

func (ar *APIRetry) Get(url string, timeRetry []int, headers map[string]string) ([]byte, error) {
	return ar.retryableRequest("GET", url, nil, timeRetry, headers)
}

func (ar *APIRetry) Post(url string, body []byte, timeRetry []int, headers map[string]string) ([]byte, error) {
	return ar.retryableRequest("POST", url, body, timeRetry, headers)
}

func (ar *APIRetry) Put(url string, body []byte, timeRetry []int, headers map[string]string) ([]byte, error) {
	return ar.retryableRequest("PUT", url, body, timeRetry, headers)
}

func (ar *APIRetry) Patch(url string, body []byte, timeRetry []int, headers map[string]string) ([]byte, error) {
	return ar.retryableRequest("PATCH", url, body, timeRetry, headers)
}

func (ar *APIRetry) Delete(url string, timeRetry []int, headers map[string]string) ([]byte, error) {
	return ar.retryableRequest("DELETE", url, nil, timeRetry, headers)
}

func (ar *APIRetry) retryableRequest(method, url string, body []byte, timeRetry []int, headers map[string]string) ([]byte, error) {
	if len(timeRetry) == 0 {
		timeRetry = []int{2,4,8,16,32}
	}

	for retry, waitTime := range timeRetry {
		fmt.Printf("Tentativa %d...\n", retry+1)

		responseBody, err := ar.getResponseBody(method, url, body, headers)
		if err != nil {
			fmt.Printf("Erro ao realizar a tentativa: %v\n", err)

			backoff := time.Duration(waitTime) * time.Second
			fmt.Printf("Esperando %v antes de tentar novamente...\n", backoff)
			time.Sleep(backoff)
			continue
		}

		return responseBody, nil
	}

	return nil, fmt.Errorf("Número máximo de tentativas atingido")
}
