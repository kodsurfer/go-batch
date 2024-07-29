package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// ExternalSystemClient представляет клиент для взаимодействия с внешней системой.
type ExternalSystemClient struct{}

// SendBatch отправляет группу запросов во внешнюю систему.
func (c *ExternalSystemClient) SendBatch(ctx context.Context, requests []BatchRequest) ([]BatchResponse, error) {
	// Симуляция нестабильной работы внешней системы.
	if len(requests) > 2 {
		return nil, errors.New("external system error")
	}

	responses := make([]BatchResponse, len(requests))
	for i, req := range requests {
		responses[i] = BatchResponse{
			ID:     req.ID,
			Result: "simulated result",
			Error:  nil,
		}
	}

	return responses, nil
}

// BatchRequest представляет запрос к внешней системе.
type BatchRequest struct {
	ID     string
	Params map[string]interface{}
}

// BatchResponse представляет ответ от внешней системы.
type BatchResponse struct {
	ID     string
	Result interface{}
	Error  error
}

// ProxyService представляет сервис прокси.
type ProxyService struct {
	client *ExternalSystemClient
	batch  chan batchItem
	wg     sync.WaitGroup
}

// batchItem представляет элемент очереди батча.
type batchItem struct {
	ctx  context.Context
	req  BatchRequest
	resp chan BatchResponse
}

// NewProxyService создает новый экземпляр ProxyService.
func NewProxyService(client *ExternalSystemClient) *ProxyService {
	svc := &ProxyService{
		client: client,
		batch:  make(chan batchItem),
	}
	svc.startBatchProcessor()
	return svc
}

// startBatchProcessor запускает процессор батчей.
func (s *ProxyService) startBatchProcessor() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case item := <-s.batch:
				s.processBatch(item)
			case <-time.After(1 * time.Second):
				s.flushBatch()
			}
		}
	}()
}

// processBatch обрабатывает один элемент батча.
func (s *ProxyService) processBatch(item batchItem) {
	batch := []BatchRequest{item.req}
	for i := 0; i < len(s.batch); i++ {
		select {
		case nextItem := <-s.batch:
			batch = append(batch, nextItem.req)
		default:
			break
		}
	}

	s.sendBatch(item.ctx, batch, item.resp)
}

// flushBatch отправляет текущий батч, если он не пустой.
func (s *ProxyService) flushBatch() {
	if len(s.batch) == 0 {
		return
	}

	batch := []BatchRequest{}
	for i := 0; i < len(s.batch); i++ {
		select {
		case item := <-s.batch:
			batch = append(batch, item.req)
		default:
			break
		}
	}

	respChan := make(chan BatchResponse, len(batch))
	s.sendBatch(context.Background(), batch, respChan)
}

// sendBatch отправляет батч во внешнюю систему.
func (s *ProxyService) sendBatch(ctx context.Context, batch []BatchRequest, respChan chan BatchResponse) {
	var responses []BatchResponse
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		responses, err = s.client.SendBatch(ctx, batch)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempt) * time.Second):
			// Экспоненциальная задержка между попытками.
		}
	}

	if err != nil {
		for range batch {
			respChan <- BatchResponse{Error: err}
		}
		return
	}

	for _, resp := range responses {
		respChan <- resp
	}
}

// HandleRequest обрабатывает запрос от клиента.
func (s *ProxyService) HandleRequest(ctx context.Context, req BatchRequest) (BatchResponse, error) {
	respChan := make(chan BatchResponse)
	select {
	case s.batch <- batchItem{ctx: ctx, req: req, resp: respChan}:
	case <-ctx.Done():
		return BatchResponse{}, ctx.Err()
	}

	select {
	case resp := <-respChan:
		return resp, resp.Error
	case <-ctx.Done():
		return BatchResponse{}, ctx.Err()
	}
}

func main() {
	client := &ExternalSystemClient{}
	service := NewProxyService(client)

	// Пример запроса от клиента.
	req := BatchRequest{
		ID: "1",
		Params: map[string]interface{}{
			"key": "value",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := service.HandleRequest(ctx, req)
	if err != nil {
		log.Fatalf("Failed to handle request: %v", err)
	}

	fmt.Printf("Response: %+v\n", resp)
}
