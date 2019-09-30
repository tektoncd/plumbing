package config

import (
	"errors"
	"flag"
)

type Config struct {
	Hostname  string
	Port      string
	Project   string
	Cluster   string
	Namespace string
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.Project, "project", "", "gke project to query")
	flag.StringVar(&c.Cluster, "cluster", "", "cluster name to query for logs")
	flag.StringVar(&c.Namespace, "namespace", "", "namespace name to query for logs")
	flag.StringVar(&c.Hostname, "hostname", "localhost", "hostname to bind to")
	flag.StringVar(&c.Port, "port", "9999", "port to bind to")
	flag.Parse()
}

func (c *Config) Validate() error {
	if c.Hostname == "" {
		return errors.New("missing hostname")
	}

	if c.Port == "" {
		return errors.New("missing port")
	}

	if c.Project == "" || c.Cluster == "" || c.Namespace == "" {
		return errors.New("missed configuration: project, cluster, namespace")
	}
	return nil
}
