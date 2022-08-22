// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"encoding/json"
	"fmt"
	"strings"

	"code.gitea.io/gitea/models"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
)

// CreateNewRef creates a new ref
func CreateNewRef(ctx *context.APIContext, doer *user_model.User, target, ref string) error {

		// Trim '--' prefix to prevent command line argument vulnerability.
		ref = strings.TrimPrefix(ref, "--")
		err := ctx.Repo.GitRepo.CreateRef(ref, target)
		if err != nil {

			errStr, _ := json.Marshal(err)
			fmt.Println(string(errStr))

			if strings.Contains(err.Error(), "is not a valid") && strings.Contains(err.Error(), " name") {
				return models.ErrInvalidRefName{
					RefName: ref,
			}
		}
	}
	return err
}
