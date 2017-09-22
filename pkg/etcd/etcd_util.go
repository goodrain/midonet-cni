package etcd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"github.com/goodrain/midonet-cni/pkg/types"
)

//CreateETCDClient 创建etcd客户端
func CreateETCDClient(conf types.ETCDConf) (client.Client, error) {
	var timeout time.Duration
	if conf.PeerTimeOut != "" {
		var err error
		timeout, err = time.ParseDuration(conf.PeerTimeOut)
		if err != nil {
			timeout = time.Second
		}
	}

	cfg := client.Config{
		Endpoints: conf.URLs,
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: timeout,
		Username:                conf.Username,
		Password:                conf.Password,
	}
	c, err := client.New(cfg)
	if err != nil {
		logrus.Error("Create etcd client error,", err.Error())
		return nil, err
	}
	return c, nil
}

//HandleError 处理错误
func HandleError(err error) error {
	if err == context.Canceled {
		logrus.Error("ctx is canceled by another routine")
		return err
	} else if err == context.DeadlineExceeded {
		logrus.Error("ctx is attached with a deadline and it exceeded")
		return err
	} else if cerr, ok := err.(*client.ClusterError); ok {
		logrus.Error("etcd cluster error.", cerr.Error())
		return cerr
	}
	logrus.Error("bad cluster endpoints, which are not etcd servers")
	return err
}

const (
	defaultTTL   = 60
	defaultTry   = 3
	deleteAction = "delete"
	expireAction = "expire"
)

// A Mutex is a mutual exclusion lock which is distributed across a cluster.
type Mutex struct {
	key    string
	id     string // The identity of the caller
	client client.Client
	kapi   client.KeysAPI
	ctx    context.Context
	ttl    time.Duration
	mutex  *sync.Mutex
}

// New creates a Mutex with the given key which must be the same
// across the cluster nodes.
func New(key string, ttl int, c client.Client) *Mutex {

	hostname, err := os.Hostname()
	if err != nil {
		return nil
	}

	if len(key) == 0 {
		return nil
	}

	if key[0] != '/' {
		key = "/" + key
	}

	if ttl < 1 {
		ttl = defaultTTL
	}

	return &Mutex{
		key:    key,
		id:     fmt.Sprintf("%v-%v-%v", hostname, os.Getpid(), time.Now().Format("20060102-15:04:05.999999999")),
		client: c,
		kapi:   client.NewKeysAPI(c),
		ctx:    context.TODO(),
		ttl:    time.Second * time.Duration(ttl),
		mutex:  new(sync.Mutex),
	}
}

// Lock locks m.
// If the lock is already in use, the calling goroutine
// blocks until the mutex is available.
func (m *Mutex) Lock() (err error) {
	m.mutex.Lock()
	for try := 1; try <= defaultTry; try++ {
		if m.lock() == nil {
			return nil
		}

		logrus.Debug("Lock node %v ERROR %v", m.key, err)
		if try < defaultTry {
			logrus.Debug("Try to lock node %v again", m.key, err)
		}
	}
	return err
}

func (m *Mutex) lock() (err error) {
	setOptions := &client.SetOptions{
		PrevExist: client.PrevNoExist,
		TTL:       m.ttl,
	}
	resp, err := m.kapi.Set(m.ctx, m.key, m.id, setOptions)
	if err == nil {
		return nil
	}
	logrus.Errorf("Create node %v failed [%v]", m.key, err)
	e, ok := err.(client.Error)
	if !ok {
		return err
	}
	if e.Code != client.ErrorCodeNodeExist {
		return err
	}
	// Get the already node's value.
	resp, err = m.kapi.Get(m.ctx, m.key, nil)
	if err != nil {
		return err
	}
	watcherOptions := &client.WatcherOptions{
		AfterIndex: resp.Index,
		Recursive:  false,
	}
	watcher := m.kapi.Watcher(m.key, watcherOptions)
	for {
		resp, err = watcher.Next(m.ctx)
		if err != nil {
			return err
		}
		if resp.Action == deleteAction || resp.Action == expireAction {
			return nil
		}
	}

}

// Unlock unlocks m.
// It is a run-time error if m is not locked on entry to Unlock.
//
// A locked Mutex is not associated with a particular goroutine.
// It is allowed for one goroutine to lock a Mutex and then
// arrange for another goroutine to unlock it.
func (m *Mutex) Unlock() (err error) {
	defer m.mutex.Unlock()
	for i := 1; i <= defaultTry; i++ {
		var resp *client.Response
		resp, err = m.kapi.Delete(m.ctx, m.key, nil)
		if err == nil {
			return nil
		}
		logrus.Errorf("Delete %v falied: %q", m.key, resp)
		e, ok := err.(client.Error)
		if ok && e.Code == client.ErrorCodeKeyNotFound {
			return nil
		}
	}
	return err
}
