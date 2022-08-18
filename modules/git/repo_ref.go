// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

// GetRefs returns all references of the repository.
func (repo *Repository) GetRefs() ([]*Reference, error) {
	return repo.GetRefsFiltered("")
}

// CreateRef creates one ref in the repository
func (repo *Repository) CreateRef(name, sha string) error {
	_, _, err := NewCommand(repo.Ctx, "update-ref", name, sha).RunStdString(&RunOpts{Dir: repo.Path})
	return err
}
