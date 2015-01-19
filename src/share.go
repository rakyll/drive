// Copyright 2015 Google Inc. All Rights Reserved.
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
	"fmt"
	"strings"
	"sync"
)

type AccountType int

const (
	UnknownAccountType = 1 << iota
	Anyone
	User
	Domain
	Group
)

type Role int

const (
	UnknownRole = 1 << iota
	Owner
	Reader
	Writer
	Commenter
)

type shareChange struct {
	emailMessage string
	emails       []string
	role         Role
	accountType  AccountType
	files        []*File
	revoke       bool
}

func (r *Role) String() string {
	switch *r {
	case Owner:
		return "owner"
	case Reader:
		return "reader"
	case Writer:
		return "writer"
	case Commenter:
		return "commenter"
	}
	return "unknown"
}

func (a *AccountType) String() string {
	switch *a {
	case Anyone:
		return "anyone"
	case User:
		return "user"
	case Domain:
		return "domain"
	case Group:
		return "group"
	}
	return "unknown"
}

func stringToRole() func(string) Role {
	roleMap := make(map[string]Role)
	roles := []Role{UnknownRole, Anyone, User, Domain, Group}
	for _, role := range roles {
		roleMap[role.String()] = role
	}
	return func(s string) Role {
		r, ok := roleMap[strings.ToLower(s)]
		if !ok {
			return UnknownRole
		}
		return r
	}
}

func stringToAccountType() func(string) AccountType {
	accountMap := make(map[string]AccountType)
	accounts := []AccountType{UnknownAccountType, Owner, Reader, Writer, Commenter}
	for _, account := range accounts {
		accountMap[account.String()] = account
	}
	return func(s string) AccountType {
		a, ok := accountMap[strings.ToLower(s)]
		if !ok {
			return UnknownAccountType
		}
		return a
	}
}

var reverseRoleResolve = stringToRole()
var reverseAccountTypeResolve = stringToAccountType()

func (g *Commands) resolveRemotePaths(relToRootPaths []string) (files []*File) {
	var wg sync.WaitGroup

	wg.Add(len(relToRootPaths))
	for _, relToRoot := range relToRootPaths {
		go func(p string, wgg *sync.WaitGroup) {
			defer wgg.Done()
			file, err := g.rem.FindByPath(p)
			if err != nil || file == nil {
				return
			}
			files = append(files, file)
		}(relToRoot, &wg)
	}
	wg.Wait()
	return files
}

func emailsToIds(g *Commands, emails []string) map[string]string {
	emailToIds := make(map[string]string)
	var wg sync.WaitGroup
	wg.Add(len(emails))
	for _, email := range emails {
		go func(email string, wgg *sync.WaitGroup) {
			defer wgg.Done()
			emailId, err := g.rem.idForEmail(email)
			if err == nil {
				emailToIds[email] = emailId
			}
		}(email, &wg)
	}
	wg.Wait()
	return emailToIds
}

func (c *Commands) Unshare() (err error) {
	return c.share(true)
}

func (c *Commands) Share() (err error) {
	return c.share(false)
}

func showPromptShareChanges(change *shareChange) bool {
	if len(change.files) < 1 {
		return false
	}
	if change.revoke {
		fmt.Printf("Revoke access for accountType: \033[92m%s\033[00m for file(s):\n", change.accountType.String())
		for _, file := range change.files {
			fmt.Println("+ ", file.Name)
		}
		fmt.Println()
		return promptForChanges()
	}

	if len(change.emails) < 1 {
		return false
	}

	fmt.Println("Message:\n\t", change.emailMessage)
	fmt.Println("Receipients:")
	for _, email := range change.emails {
		fmt.Printf("\t\033[92m+\033[00m %s\n", email)
	}

	fmt.Println("\nFile(s) to share:")
	for _, file := range change.files {
		if file == nil {
			continue
		}
		fmt.Printf("\t\033[92m+\033[00m %s\n", file.Name)
	}
	return promptForChanges()
}

func (c *Commands) playShareChanges(change *shareChange) error {
	if !showPromptShareChanges(change) {
		return nil
	}
	for _, file := range change.files {
		if change.revoke {
			if err := c.rem.deletePermissions(file.Id, change.accountType); err != nil {
				return fmt.Errorf("%s: %v", file.Name, err)
			}
		}

		for _, email := range change.emails {
			_, err := c.rem.insertPermissions(file.Id, email, change.emailMessage, change.role, change.accountType)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Commands) share(revoke bool) (err error) {
	files := c.resolveRemotePaths(c.opts.Sources)

	var role Role
	var accountType AccountType
	var emails []string
	var emailMessage string

	meta := *c.opts.Meta
	if meta != nil {
		emailList, eOk := meta["emails"]
		if eOk {
			emails = emailList
			if false {
				emailIdMap := emailsToIds(c, emailList)
				fmt.Println(emailIdMap)
			}
		}

		roleList, rOk := meta["role"]
		if rOk && len(roleList) >= 1 {
			role = reverseRoleResolve(roleList[0])
		}
		accountTypeList, aOk := meta["accountType"]
		if aOk {
			accountType = reverseAccountTypeResolve(accountTypeList[0])
		}

		emailMessageList, emOk := meta["emailMessage"]
		if emOk && len(emailMessageList) >= 1 {
			emailMessage = strings.Join(emailMessageList, "\n")
		}
	}

	change := shareChange{
		accountType:  accountType,
		emailMessage: emailMessage,
		emails:       emails,
		files:        files,
		revoke:       revoke,
		role:         role,
	}

	return c.playShareChanges(&change)
}
