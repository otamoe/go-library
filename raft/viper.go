package libraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	dconfig "github.com/lni/dragonboat/v3/config"
	"github.com/spf13/viper"
)

func ViperDeploymentID() (out OutNodeHostConfig) {
	out.Config = func(nhc dconfig.NodeHostConfig) (dconfig.NodeHostConfig, error) {
		nhc.DeploymentID = viper.GetUint64("raft.deploymentID")
		return nhc, nil
	}
	return
}

func ViperAddress() (out OutNodeHostConfig) {
	out.Config = func(nhc dconfig.NodeHostConfig) (dconfig.NodeHostConfig, error) {
		nhc.RaftAddress = viper.GetString("raft.address")
		return nhc, nil
	}
	return
}
func ViperListenAddress() (out OutNodeHostConfig) {
	out.Config = func(nhc dconfig.NodeHostConfig) (dconfig.NodeHostConfig, error) {
		nhc.ListenAddress = viper.GetString("raft.listenAddress")
		return nhc, nil
	}
	return
}

func ViperTLS() (out OutNodeHostConfig) {
	// tls 配置
	if len(viper.GetString("raft.tls.ca")) != 0 || len(viper.GetString("raft.tls.cert")) != 0 || len(viper.GetString("raft.tls.key")) != 0 {

		out.Config = func(nhc dconfig.NodeHostConfig) (outnhc dconfig.NodeHostConfig, err error) {
			nhc.MutualTLS = true

			// ca 文件
			if nhc.CAFile == "" && len(viper.GetString("raft.tls.ca")) != 0 {
				nhc.CAFile = path.Join(os.TempDir(), fmt.Sprintf("raft-tls-ca-%d", AddrRaftNodeIDP(nhc.RaftAddress)))
				if err = ioutil.WriteFile(nhc.CAFile, []byte(viper.GetString("raft.tls.ca")), 0755); err != nil {
					return
				}
			}

			// cert 文件
			if nhc.CertFile == "" && len(viper.GetString("raft.tls.cert")) != 0 {
				nhc.CertFile = path.Join(os.TempDir(), fmt.Sprintf("raft-tls-cert-%d", AddrRaftNodeIDP(nhc.RaftAddress)))
				if err = ioutil.WriteFile(nhc.CertFile, []byte(viper.GetString("raft.tls.cert")), 0755); err != nil {
					panic(err)
				}
			}

			// key 文件
			if nhc.KeyFile == "" && len(viper.GetString("raft.tls.key")) != 0 {
				nhc.KeyFile = path.Join(os.TempDir(), fmt.Sprintf("raft-tls-key-%d", AddrRaftNodeIDP(nhc.RaftAddress)))
				if err = ioutil.WriteFile(nhc.KeyFile, []byte(viper.GetString("raft.tls.key")), 0755); err != nil {
					panic(err)
				}
			}
			outnhc = nhc
			return
		}
		return
	}

	out.Config = func(nhc dconfig.NodeHostConfig) (dconfig.NodeHostConfig, error) {
		return nhc, nil
	}
	return
}
