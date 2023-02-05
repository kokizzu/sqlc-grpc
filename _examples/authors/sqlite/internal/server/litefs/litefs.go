// Code generated by sqlc-grpc (https://github.com/walterwanderley/sqlc-grpc).

package litefs

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/superfly/litefs"
	"github.com/superfly/litefs/fuse"
	litehttp "github.com/superfly/litefs/http"
	litefsraft "github.com/walterwanderley/litefs-raft"
	"go.uber.org/zap"
)

type LiteFS struct {
	raft           *raft.Raft
	store          *litefs.Store
	fileSystem     *fuse.FileSystem
	httpServer     *litehttp.Server
	redirectTarget RedirectTarget
}

func (lfs *LiteFS) ReadyCh() chan struct{} {
	return lfs.store.ReadyCh()
}

type node struct {
	ID       string `json:"id"`
	Addr     string `json:"addr"`
	ReadOnly bool   `json:"readOnly"`
}

func (lfs *LiteFS) ClusterHandler(w http.ResponseWriter, r *http.Request) {
	if lfs.raft == nil {
		http.Error(w, "cluster is running in static mode", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var nodeInfo node
		err := json.NewDecoder(r.Body).Decode(&nodeInfo)
		if err != nil {
			http.Error(w, "cannot parse node info", http.StatusBadRequest)
			return
		}
		if nodeInfo.ReadOnly {
			err = lfs.raft.AddNonvoter(raft.ServerID(nodeInfo.ID), raft.ServerAddress(nodeInfo.Addr), 0, 0).Error()
		} else {
			err = lfs.raft.AddVoter(raft.ServerID(nodeInfo.ID), raft.ServerAddress(nodeInfo.Addr), 0, 0).Error()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		var nodeID string
		i := strings.LastIndex(r.URL.Path, "/")
		if i < len(r.URL.Path) {
			nodeID = r.URL.Path[i+1:]
		}
		if nodeID == "" {
			http.Error(w, "node ID is required", http.StatusBadRequest)
			return
		}
		future := lfs.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
		if err := future.Error(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodGet:
		if strings.HasSuffix(r.URL.Path, "leader") {
			addr, id := lfs.raft.LeaderWithID()
			if string(id) == "" {
				http.Error(w, "leader not found", http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(node{
				ID:   string(id),
				Addr: string(addr),
			})
			return
		}
		configFuture := lfs.raft.GetConfiguration()
		if err := configFuture.Error(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		servers := configFuture.Configuration().Servers
		nodesList := make([]node, len(servers))
		for i, server := range servers {
			nodesList[i] = node{
				ID:       string(server.ID),
				Addr:     string(server.Address),
				ReadOnly: server.Suffrage == raft.Nonvoter,
			}
		}
		json.NewEncoder(w).Encode(nodesList)

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

}

func (lfs *LiteFS) Close() (err error) {
	if e := lfs.httpServer.Close(); err == nil {
		err = e
	}
	if e := lfs.fileSystem.Unmount(); err == nil {
		err = e
	}
	if e := lfs.store.Close(); err == nil {
		err = e
	}

	return
}

func Start(log *zap.Logger, cfg Config) (*LiteFS, error) {
	store := litefs.NewStore(cfg.ConfigDir, cfg.Candidate)
	store.Client = litehttp.NewClient()

	var (
		leaser         litefs.Leaser
		r              *raft.Raft
		redirectTarget RedirectTarget
	)
	if cfg.RaftPort > 0 {
		fsm := litefsraft.NewFSM()
		r, err := startRaft(cfg, fsm)
		if err != nil {
			return nil, fmt.Errorf("cannot start RAFT: %w", err)
		}
		localInfo := litefsraft.PrimaryRedirectInfo{
			PrimaryInfo: litefs.PrimaryInfo{
				Hostname:     cfg.Hostname,
				AdvertiseURL: cfg.AdvertiseURL,
			},
			RedirectURL: cfg.RedirectURL,
		}
		leaser = litefsraft.New(r, localInfo, fsm, 10*time.Second)
		redirectTarget = func() string {
			return fsm.RedirectURL()
		}
	} else {
		leaser = litefs.NewStaticLeaser(cfg.Candidate, cfg.Hostname, cfg.AdvertiseURL)
		redirectTarget = func() string {
			return cfg.RedirectURL
		}
	}

	store.Leaser = leaser

	server := litehttp.NewServer(store, fmt.Sprintf(":%d", cfg.Port))
	if err := server.Listen(); err != nil {
		return nil, fmt.Errorf("cannot open http server: %w", err)
	}
	server.Serve()

	if err := store.Open(); err != nil {
		return nil, fmt.Errorf("cannot open store: %w", err)
	}
	fsys := fuse.NewFileSystem(cfg.MountDir, store)
	if err := fsys.Mount(); err != nil {
		return nil, fmt.Errorf("cannot open file system: %s", err)
	}
	store.Invalidator = fsys

	return &LiteFS{
		raft:           r,
		store:          store,
		fileSystem:     fsys,
		httpServer:     server,
		redirectTarget: redirectTarget,
	}, nil
}

func startRaft(cfg Config, fsm raft.FSM) (*raft.Raft, error) {
	dir := filepath.Join(cfg.ConfigDir, "raft_"+cfg.Hostname)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("cannot create config directory: %w", err)
	}

	store, err := raftboltdb.NewBoltStore(filepath.Join(dir, "bolt"))
	if err != nil {
		return nil, fmt.Errorf("cannot create bolt store: %w", err)
	}

	snapshots, err := raft.NewFileSnapshotStore(filepath.Join(dir, "snapshot"), 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("cannot create snapshot store: %w", err)
	}

	raftCfg := raft.DefaultConfig()
	raftCfg.LocalID = raft.ServerID(cfg.Hostname)

	addr, err := net.ResolveTCPAddr("tcp", cfg.RaftAdvertiseAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve raft advertise address: %w", err)
	}
	transport, err := raft.NewTCPTransport(cfg.RaftAdvertiseAddress, addr, 9, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("cannot create raft TCP transport: %w", err)
	}

	r, err := raft.NewRaft(raftCfg, fsm, store, store, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("cannot create raft: %w", err)
	}

	if cfg.BootstrapCluster && r.LastIndex() == 0 {
		var nodes []raft.Server
		nodes = append(nodes, raft.Server{
			ID:      raft.ServerID(cfg.Hostname),
			Address: raft.ServerAddress(cfg.RaftAdvertiseAddress),
		})
		if cfg.Members != "" {
			for _, node := range strings.Split(cfg.Members, ",") {
				id, addr, ok := strings.Cut(node, "=")
				if !ok {
					log.Fatal("invalid -members parameter")
				}
				addr = strings.TrimSpace(addr)
				var suffrage raft.ServerSuffrage
				if strings.HasSuffix(addr, ":ro") {
					suffrage = raft.Nonvoter
					addr = strings.TrimSuffix(addr, ":ro")
				} else {
					suffrage = raft.Voter
				}
				nodes = append(nodes, raft.Server{
					ID:       raft.ServerID(strings.TrimSpace(id)),
					Address:  raft.ServerAddress(addr),
					Suffrage: suffrage,
				})
			}
		}
		r.BootstrapCluster(
			raft.Configuration{
				Servers: nodes,
			},
		)
	}

	return r, nil
}

type Config struct {
	Port                 int
	ConfigDir            string
	MountDir             string
	Hostname             string
	AdvertiseURL         string
	Candidate            bool
	RaftPort             int
	RaftAdvertiseAddress string
	Members              string
	BootstrapCluster     bool
	RedirectURL          string
}

func (c Config) Validate() error {
	if c.MountDir == "" {
		return nil
	}

	if c.ConfigDir == "" {
		return fmt.Errorf("--litefs-config-dir is required")
	}

	if c.Candidate && c.AdvertiseURL == "" {
		return fmt.Errorf("-litefs-advertise-url is required for candidate instance")
	}
	return nil
}

func SetFlags(cfg *Config) {
	flag.StringVar(&cfg.ConfigDir, "litefs-config-dir", "", "LiteFS config directory")
	flag.StringVar(&cfg.MountDir, "litefs-mount-dir", "", "LiteFS fuse mount dir")
	flag.StringVar(&cfg.Hostname, "litefs-hostname", "", "LiteFS instance identify")
	flag.StringVar(&cfg.AdvertiseURL, "litefs-advertise-url", "", "LiteFS advertise URL")
	flag.IntVar(&cfg.Port, "litefs-port", 20202, "LiteFS server port")
	flag.IntVar(&cfg.RaftPort, "litefs-raft-port", 0, "Raft port")
	flag.StringVar(&cfg.RaftAdvertiseAddress, "litefs-raft-addr", "", "Raft advertise address")
	flag.StringVar(&cfg.Members, "litefs-members", "", "Comma separated list of clusters members. Example: -litefs-members \"Hostname1=RaftAddress1, Hostname2=RaftAddress2\"")
	flag.BoolVar(&cfg.Candidate, "litefs-candidate", true, "Specifies whether the node can become the primary")
	flag.BoolVar(&cfg.BootstrapCluster, "litefs-bootstrap-cluster", true, "Bootstrap the cluster")
	flag.StringVar(&cfg.RedirectURL, "litefs-redirect", "", "Redirect requests to this URL if this instance is the leader")
}
