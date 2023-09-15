package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	log "k8s.io/klog/v2"

	"github.com/vanus-labs/gateway-proxy/db"
	"github.com/vanus-labs/gateway-proxy/models"
	"github.com/vanus-labs/gateway-proxy/monitor"
	"github.com/vanus-labs/gateway-proxy/region"
)

// forward http://vanus-gateway.vanus:8081/namespaces/default/eventbus/p0qcb5te/events to vanus-core gateway

var (
	RouteCache = make(map[string]string, 0)
)

type Config struct {
	Port    int            `yaml:"port"`
	DB      db.Config      `yaml:"mongodb"`
	Monitor monitor.Config `yaml:"monitor"`
	Region  models.Region  `yaml:"region"`
}

func ParseConfig(c *Config) error {
	file := os.Getenv("CONFIG_FILE")
	if file == "" {
		file = "./config/config.yml"
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bytes, c)
}

func main() {
	ctx := context.Background()
	var c Config
	err := ParseConfig(&c)
	if err != nil {
		panic(err)
	}

	cli, err := db.Init(ctx, c.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize mongodb client: %s", err))
	}
	defer func() {
		_ = cli.Disconnect(ctx)
	}()

	monitor.Init(ctx, c.Monitor)

	err = region.Init(ctx, c.Region)
	if err != nil {
		panic(err)
	}

	proxyServer := &ProxyServer{
		regions: region.GetAllRegionInfo(ctx),
	}
	http.HandleFunc("/", proxyServer.handleRequest)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil))
}

type ProxyServer struct {
	regions []models.RegionInfo
}

func (p *ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	var requested string
	body, _ := io.ReadAll(r.Body)
	if endpoint, ok := RouteCache[r.URL.Path]; ok {
		resp, err := func() (*http.Response, error) {
			req, err := http.NewRequest(r.Method, endpoint+r.URL.String(), io.NopCloser(bytes.NewBuffer(body)))
			if err != nil {
				log.Infof("failed to create proxy request: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "failed to create proxy request")
				return nil, err
			}
			copyHeaders(r.Header, req.Header)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				copyHeaders(resp.Header, w.Header())
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Errorf("failed to read proxy response body: %v\n", err)
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "failed to read proxy response body")
					return resp, err
				}
				w.WriteHeader(resp.StatusCode)
				w.Write(body)
				return resp, nil
			}
			message := fmt.Sprintf("【Sends Failure】Sending event to %s of %s cluster failed", getEventbus(r.URL.Path), isDefaultCluster(endpoint))
			monitor.SendAlarm(context.Background(), message)
			if resp == nil {
				log.Errorf("proxy request to cluster %s failed, url_path: %s, err: %+v\n", endpoint, r.URL.Path, err)
			} else {
				log.Errorf("proxy request to cluster %s failed, url_path: %s, resp_code: %d, err: %+v\n", endpoint, r.URL.Path, resp.StatusCode, err)
			}
			return resp, err
		}()
		if resp != nil {
			if err == nil && resp.StatusCode == http.StatusOK {
				return
			} else {
				requested = endpoint
			}
		}
	}
	for _, region := range p.regions {
		endpoint := region.Gateway.EndpointStr()
		if endpoint == requested {
			continue
		}
		req, err := http.NewRequest(r.Method, endpoint+r.URL.String(), io.NopCloser(bytes.NewBuffer(body)))
		if err != nil {
			log.Infof("failed to create proxy request: %v\n", err)
			continue
		}
		copyHeaders(r.Header, req.Header)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			copyHeaders(resp.Header, w.Header())
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Errorf("failed to read proxy response body: %v\n", err)
				continue
			}
			w.WriteHeader(resp.StatusCode)
			w.Write(body)
			if len(p.regions) > 1 {
				RouteCache[r.URL.Path] = endpoint
			}
			return
		}
		message := fmt.Sprintf("【Sends Failure】Sending event to %s of %s cluster failed", getEventbus(r.URL.Path), isDefaultCluster(endpoint))
		monitor.SendAlarm(context.Background(), message)
		if resp == nil {
			log.Errorf("proxy request to cluster %s failed, url_path: %s, err: %+v\n", endpoint, r.URL.Path, err)
		} else {
			log.Errorf("proxy request to cluster %s failed, url_path: %s, resp_code: %d, err: %+v\n", endpoint, r.URL.Path, resp.StatusCode, err)
		}
	}
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "all proxy requests failed")
}

func copyHeaders(src http.Header, dst http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isDefaultCluster(endpoint string) string {
	if strings.Contains(endpoint, "default") {
		return "default"
	}
	if strings.Contains(endpoint, "standby") {
		return "standby"
	}
	return endpoint
}

func getEventbus(path string) string {
	subPaths := strings.Split(path, "/")
	subPathNum := len(subPaths)
	if subPathNum >= 2 {
		return subPaths[subPathNum-2]
	}
	return path
}
