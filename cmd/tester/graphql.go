package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// GraphQlApiConfig -
type GraphQlApiConfig struct {
	Url               string `yaml:"url" validate:"required,url"`
	RequestsPerSecond int    `yaml:"requests_per_seconds" validate:"required,min=1"`
}

// ActualDomainsResponse -
type ActualDomainsResponse struct {
	Data struct {
		ActualDomains []ActualDomain `json:"actual_domains"`
	} `json:"data"`
}

// GraphQlRequest -
type GraphQlRequest struct {
	OperationName string         `json:"operationName"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
}

// ActualDomain -
type ActualDomain struct {
	ID      string    `json:"id"`
	Domain  string    `json:"domain"`
	Address string    `json:"address"`
	Expiry  time.Time `json:"expiry"`
}

// GraphQlApi -
type GraphQlApi struct {
	client    *http.Client
	baseURL   string
	rateLimit *rate.Limiter
}

// NewGraphQlApi -
func NewGraphQlApi(cfg GraphQlApiConfig) GraphQlApi {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	client := &http.Client{
		Transport: t,
	}
	api := GraphQlApi{
		client:    client,
		baseURL:   cfg.Url,
		rateLimit: rate.NewLimiter(rate.Every(time.Second/time.Duration(cfg.RequestsPerSecond)), cfg.RequestsPerSecond),
	}

	return api
}

func (api GraphQlApi) post(ctx context.Context, body GraphQlRequest, output any) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, api.baseURL, buf)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")

	if api.rateLimit != nil {
		if err := api.rateLimit.Wait(ctx); err != nil {
			return err
		}
	}

	response, err := api.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("invalid status code: %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(output); err != nil {
		return err
	}

	return nil
}

const actualDomainsRequest = `query GetActualDomains ($limit: Int!, $offset:Int!) {
    actual_domains(order_by: {id: asc}, limit: $limit, offset: $offset) {
      id
      domain
      address
      expiry
    }
  }
  `

// ActualDomains -
func (api GraphQlApi) ActualDomains(ctx context.Context, limit, offset int) ([]ActualDomain, error) {
	body := GraphQlRequest{
		OperationName: "GetActualDomains",
		Query:         actualDomainsRequest,
		Variables: map[string]any{
			"limit":  limit,
			"offset": offset,
		},
	}
	var response ActualDomainsResponse
	if err := api.post(ctx, body, &response); err != nil {
		return nil, err
	}
	return response.Data.ActualDomains, nil
}
