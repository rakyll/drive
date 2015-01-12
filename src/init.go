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

package drive

import (
	"os"
)

func (g *Commands) Init() (err error) {
	var refresh string

	g.context.ClientId = os.Getenv("GOOGLE_API_CLIENT_ID")
	g.context.ClientSecret = os.Getenv("GOOGLE_API_CLIENT_SECRET")
	if g.context.ClientId == "" || g.context.ClientSecret == "" {
		g.context.ClientId = "354790962074-7rrlnuanmamgg1i4feed12dpuq871bvd.apps.googleusercontent.com"
		g.context.ClientSecret = "RHjKdah8RrHFwu6fcc0uEVCw"
	}

	if refresh, err = RetrieveRefreshToken(g.context); err != nil {
		return
	}
	g.context.RefreshToken = refresh
	err = g.context.Write()
	return
}
