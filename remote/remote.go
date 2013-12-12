// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rakyll/gd/config"
	"github.com/rakyll/gd/third_party/code.google.com/p/goauth2/oauth"
	drive "github.com/rakyll/gd/third_party/code.google.com/p/google-api-go-client/drive/v2"
	"github.com/rakyll/gd/types"
)

const (
	// Google OAuth 2.0 service URLs
	GoogleOAuth2AuthURL  = "https://accounts.google.com/o/oauth2/auth"
	GoogleOAuth2TokenURL = "https://accounts.google.com/o/oauth2/token"

	// OAuth 2.0 OOB redirect URL for authorization.
	RedirectURL = "urn:ietf:wg:oauth:2.0:oob"

	// OAuth 2.0 full Drive scope used for authorization.
	DriveScope = "https://www.googleapis.com/auth/drive"

	// OAuth 2.0 access type for offline/refresh access.
	AccessType = "offline"
)

var (
	ErrPathNotExists = errors.New("remote path doesn't exist")
)

type Remote struct {
	transport *oauth.Transport
	service   *drive.Service
}

func New(context *config.Context) *Remote {
	transport := newTransport(context)
	service, _ := drive.New(transport.Client())
	return &Remote{service: service, transport: transport}
}

func (r *Remote) FindById(id string) (file *types.File, err error) {
	req := r.service.Files.Get(id)
	var f *drive.File
	if f, err = req.Do(); err != nil {
		return
	}
	return types.NewRemoteFile(f), nil
}

func (r *Remote) FindByPath(p string) (file *types.File, err error) {
	if p == "/" {
		return r.FindById("root")
	}
	parts := strings.Split(p, "/")
	return r.findByPathRecv("root", parts[1:])
}

func (r *Remote) FindByParentId(parentId string) (files []*types.File, err error) {
	req := r.service.Files.List()
	// TODO: use field selectors
	req.Q(fmt.Sprintf("'%s' in parents and trashed=false", parentId))
	results, err := req.Do()
	if err != nil {
		return
	}
	for _, f := range results.Items {
		if !strings.HasPrefix(f.Title, ".") { // ignore hidden files
			files = append(files, types.NewRemoteFile(f))
		}
	}
	return
}

func (r *Remote) Trash(id string) error {
	_, err := r.service.Files.Trash(id).Do()
	return err
}

func (r *Remote) Download(id string) (io.ReadCloser, error) {
	resp, err := r.transport.Client().Get("https://googledrive.com/host/" + id)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.Body, err
	}
	return resp.Body, nil
}

func (r *Remote) Upsert(parentId string, file *types.File, body io.Reader) (f *types.File, err error) {
	uploaded := &drive.File{
		Title:   file.Name,
		Parents: []*drive.ParentReference{&drive.ParentReference{Id: parentId}},
	}
	if file.IsDir {
		uploaded.MimeType = "application/vnd.google-apps.folder"
	}

	if file.Id == "" {
		req := r.service.Files.Insert(uploaded)
		if !file.IsDir && body != nil {
			req = req.Media(body)
		}
		if uploaded, err = req.Do(); err != nil {
			return
		}
		return types.NewRemoteFile(uploaded), nil
	}
	// update the existing
	req := r.service.Files.Update(file.Id, uploaded)
	if !file.IsDir && body != nil {
		req = req.Media(body)
	}
	if uploaded, err = req.Do(); err != nil {
		return
	}
	return types.NewRemoteFile(uploaded), nil
}

func (r *Remote) findByPathRecv(parentId string, p []string) (file *types.File, err error) {
	// find the file or directory under parentId and titled with p[0]
	req := r.service.Files.List()
	// TODO: use field selectors
	req.Q(fmt.Sprintf("'%s' in parents and title = '%s' and trashed=false", parentId, p[0]))
	files, err := req.Do()
	if len(files.Items) < 1 || err != nil {
		return nil, ErrPathNotExists
	}
	file = types.NewRemoteFile(files.Items[0])
	if len(p) == 1 {
		return file, nil
	}
	return r.findByPathRecv(file.Id, p[1:])
}

func newAuthConfig(context *config.Context) *oauth.Config {
	return &oauth.Config{
		ClientId:     context.ClientId,
		ClientSecret: context.ClientSecret,
		AuthURL:      GoogleOAuth2AuthURL,
		TokenURL:     GoogleOAuth2TokenURL,
		RedirectURL:  RedirectURL,
		AccessType:   AccessType,
		Scope:        DriveScope,
	}
}

func newTransport(context *config.Context) *oauth.Transport {
	return &oauth.Transport{
		Config:    newAuthConfig(context),
		Transport: http.DefaultTransport,
		Token: &oauth.Token{
			RefreshToken: context.RefreshToken,
			Expiry:       time.Now(),
		},
	}
}
