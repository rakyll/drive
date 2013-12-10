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

package god

import (
	"fmt"
)

// Creates a .god directory at the dest directory
// and initializes directory with required config files. Runs auth
// automatically to retrieve required auths.
// If there already exists such a directory, exists with error.
func (g *God) Init(absPath string) {
	if g.context != nil && g.context.AbsPath == absPath {
		fmt.Println(ErrContextExists)
		return // TODO: exit with error status
	}
}
