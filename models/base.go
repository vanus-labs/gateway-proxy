package models

import (
	"errors"
	"fmt"
)

type RegionInfo struct {
	Name                   Region   `bson:"name"`
	Provider               string   `bson:"provider"`
	Location               string   `bson:"location"`
	Token                  string   `bson:"token"`
	IsDefault              bool     `bson:"is_default"`
	Gateway                Endpoint `bson:"gateway"`
	Operator               Endpoint `bson:"operator"`
	Prometheus             Endpoint `bson:"prometheus"`
	ExternalDNS            string   `bson:"external_dns"`
	IntegrationExternalDNS string   `bson:"integration_external_dns"`
}

func (r RegionInfo) Validate() error {
	if r.Name == Region("") {
		return errors.New("region name is empty")
	}
	err := r.Gateway.Validate()
	if err != nil {
		return err
	}
	err = r.Operator.Validate()
	if err != nil {
		return err
	}
	err = r.Prometheus.Validate()
	if err != nil {
		return err
	}
	return nil
}

type Endpoint struct {
	Host string `bson:"host"`
	Port Port   `bson:"port"`
}

type Port struct {
	Http int `bson:"http"`
	Grpc int `bson:"grpc"`
}

func (e Endpoint) Validate() error {
	if e.Host == "" {
		return errors.New("endpoint host can't be empty")
	}
	if e.Port.Http == 0 {
		return errors.New("region http port is empty")
	}
	return nil
}

func (e Endpoint) EndpointStr() string {
	return fmt.Sprintf("http://%s:%d", e.Host, e.Port.Http)
}
