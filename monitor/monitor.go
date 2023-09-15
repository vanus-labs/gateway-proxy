package monitor

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"gopkg.in/resty.v1"
	log "k8s.io/klog/v2"
)

var (
	cfg    Config
	client *resty.Client
	once   sync.Once
)

type Config struct {
	Enable     bool   `yaml:"enable"`
	WebhookUrl string `yaml:"webhook_url"`
}

type Alarm struct {
	Message string `json:"message" yaml:"message"`
}

func Init(ctx context.Context, c Config) {
	once.Do(func() {
		cfg = c
		client = resty.New()
		log.Infof("the monitoring alarm function has been enabled, webhook url: %s\n", c.WebhookUrl)
	})
}

func SendAlarm(ctx context.Context, message string) error {
	req := &Alarm{
		Message: message,
	}
	resp, err := client.R().SetBody(req).Post(cfg.WebhookUrl)
	if err == handleHTTPResponse(ctx, resp, err) {
		return err
	}
	return nil
}

func handleHTTPResponse(ctx context.Context, res *resty.Response, err error) error {
	if err != nil {
		log.Warningf("HTTP request failed, err: %+v\n", err)
		return err
	}
	if res.StatusCode() != http.StatusOK {
		log.Warningf("HTTP response not 200 failed, status_code: %d, body: %s\n", res.StatusCode(), res.Body())
		return errors.New(string(res.Body()))
	}
	return nil
}
