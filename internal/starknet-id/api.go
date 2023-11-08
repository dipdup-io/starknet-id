package starknetid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// ApiConfig -
type ApiConfig struct {
	Url               string `validate:"required,url"   yaml:"url"`
	RequestsPerSecond int    `validate:"required,min=1" yaml:"requests_per_seconds"`
}

// ApiError -
type ApiError struct {
	Error string `json:"error"`
}

// DomainToAddrResponse -
type DomainToAddrResponse struct {
	Addr         string `json:"addr"`
	DomainExpiry int    `json:"domain_expiry"`
}

// AddrToDomainResponse -
type AddrToDomainResponse struct {
	Domain       string `json:"domain"`
	DomainExpiry int    `json:"domain_expiry"`
}

// Api -
type Api struct {
	client    *http.Client
	baseURL   string
	rateLimit *rate.Limiter
}

// NewApi -
func NewApi(cfg ApiConfig) Api {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	client := &http.Client{
		Transport: t,
	}
	api := Api{
		client:    client,
		baseURL:   cfg.Url,
		rateLimit: rate.NewLimiter(rate.Every(time.Second/time.Duration(cfg.RequestsPerSecond)), cfg.RequestsPerSecond),
	}

	return api
}

func (api Api) get(ctx context.Context, requestUrl string, output any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestUrl, nil)
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

	decoder := json.NewDecoder(response.Body)

	if response.StatusCode != http.StatusOK {
		var apiErr ApiError
		if err := decoder.Decode(&apiErr); err != nil {
			return errors.Wrapf(err, "invalid status code: %d", response.StatusCode)
		}
		return errors.Errorf("invalid status code (%s): %d", apiErr.Error, response.StatusCode)
	}

	if err := decoder.Decode(output); err != nil {
		return err
	}

	return nil
}

// DomainToAddress -
func (api Api) DomainToAddress(ctx context.Context, domain string) (resp DomainToAddrResponse, err error) {
	url, err := url.JoinPath(api.baseURL, "api/indexer/domain_to_addr")
	if err != nil {
		return resp, err
	}
	url = fmt.Sprintf("%s?domain=%s", url, domain)
	err = api.get(ctx, url, &resp)
	return
}

// AddressToDomain -
func (api Api) AddressToDomain(ctx context.Context, address string) (resp AddrToDomainResponse, err error) {
	url, err := url.JoinPath(api.baseURL, "api/indexer/addr_to_domain")
	if err != nil {
		return resp, err
	}
	url = fmt.Sprintf("%s?addr=%s", url, address)
	err = api.get(ctx, url, &resp)
	return
}
