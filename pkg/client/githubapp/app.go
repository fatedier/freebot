package githubapp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

type key int

const (
	installIDKey key = 0
)

func WithInstallID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, installIDKey, id)
}

type GithubAppInstallTransport struct {
	tr                http.RoundTripper
	appID             int
	privateKey        []byte
	installTransports map[int]http.RoundTripper
	mu                sync.RWMutex

	doingUpdateID map[int]struct{}
	doingMu       sync.Mutex
}

func NewGithubAppInstallTransport(tr http.RoundTripper, appID int, privateKeyFile string) (*GithubAppInstallTransport, error) {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("could not read private key: %s", err)
	}

	appTr, err := ghinstallation.NewAppsTransport(tr, appID, privateKey)
	if err != nil {
		return nil, err
	}
	githubCli := github.NewClient(&http.Client{Transport: appTr})
	installs, _, err := githubCli.Apps.ListInstallations(context.Background(), &github.ListOptions{
		Page:    1,
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	out := &GithubAppInstallTransport{
		tr:                tr,
		appID:             appID,
		privateKey:        privateKey,
		installTransports: make(map[int]http.RoundTripper),
		doingUpdateID:     make(map[int]struct{}),
	}
	for _, install := range installs {
		id := int(install.GetID())
		fmt.Printf("install id [%d]\n", id)
		insTr, err := ghinstallation.New(tr, appID, id, privateKey)
		if err != nil {
			return nil, fmt.Errorf("get transport for install ID %d error: %v", id, err)
		}
		insTr.BaseURL = "https://api.github.com/app"
		out.installTransports[id] = insTr
	}
	return out, nil
}

func (tr *GithubAppInstallTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	installID, ok := req.Context().Value(installIDKey).(int)
	if !ok {
		return nil, fmt.Errorf("no installID")
	}

	tr.mu.RLock()
	insTr, ok := tr.installTransports[installID]
	tr.mu.RUnlock()
	if !ok {
		tr.doingMu.Lock()
		_, ok = tr.doingUpdateID[installID]
		if !ok {
			tr.doingUpdateID[installID] = struct{}{}
			go tr.updateInstallTransportWorker(installID)
		}
		tr.doingMu.Unlock()
		return nil, fmt.Errorf("no corresponding install transport")
	}

	return insTr.RoundTrip(req)
}

func (tr *GithubAppInstallTransport) updateInstallTransportWorker(installID int) {
	insTr, err := ghinstallation.New(tr.tr, tr.appID, installID, tr.privateKey)
	if err != nil {
		tr.doingMu.Lock()
		delete(tr.doingUpdateID, installID)
		tr.doingMu.Unlock()
		return
	}

	insTr.BaseURL = "https://api.github.com/app"
	tr.mu.Lock()
	tr.doingMu.Lock()
	tr.installTransports[installID] = insTr
	delete(tr.doingUpdateID, installID)
	tr.doingMu.Unlock()
	tr.mu.Unlock()
}
