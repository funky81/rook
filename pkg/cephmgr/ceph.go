package cephmgr

import (
	"path"

	ctx "golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"

	"github.com/quantum/castle/pkg/cephmgr/client"
	"github.com/quantum/castle/pkg/clusterd"
	"github.com/quantum/castle/pkg/util"
)

const (
	cephName         = "ceph"
	cephKey          = "/castle/services/ceph"
	cephInstanceName = "default"
	desiredKey       = "desired"
	appliedKey       = "applied"
)

type ClusterInfo struct {
	FSID          string
	MonitorSecret string
	AdminSecret   string
	Name          string
	Monitors      map[string]*CephMonitorConfig
}

// create a new ceph service
func NewCephService(factory client.ConnectionFactory, devices string, forceFormat bool, location string) *clusterd.ClusterService {
	return &clusterd.ClusterService{
		Name:   cephName,
		Leader: newCephLeader(factory),
		Agents: []clusterd.ServiceAgent{
			&monAgent{factory: factory},
			newOSDAgent(factory, devices, forceFormat, location),
		},
	}
}

// attempt to load any previously created and saved cluster info
func LoadClusterInfo(etcdClient etcd.KeysAPI) (*ClusterInfo, error) {
	resp, err := etcdClient.Get(ctx.Background(), path.Join(cephKey, "fsid"), nil)
	if err != nil {
		if util.IsEtcdKeyNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	fsid := resp.Node.Value

	name, err := GetClusterName(etcdClient)
	if err != nil {
		return nil, err
	}

	secretsKey := path.Join(cephKey, "_secrets")

	resp, err = etcdClient.Get(ctx.Background(), path.Join(secretsKey, "monitor"), nil)
	if err != nil {
		return nil, err
	}
	monSecret := resp.Node.Value

	resp, err = etcdClient.Get(ctx.Background(), path.Join(secretsKey, "admin"), nil)
	if err != nil {
		return nil, err
	}
	adminSecret := resp.Node.Value

	cluster := &ClusterInfo{
		FSID:          fsid,
		MonitorSecret: monSecret,
		AdminSecret:   adminSecret,
		Name:          name,
	}

	// Get the monitors that have been applied in a previous orchestration
	cluster.Monitors, err = GetDesiredMonitors(etcdClient)

	return cluster, nil
}

func GetClusterName(etcdClient etcd.KeysAPI) (string, error) {
	resp, err := etcdClient.Get(ctx.Background(), path.Join(cephKey, "name"), nil)
	if err != nil {
		return "", err
	}
	return resp.Node.Value, nil
}