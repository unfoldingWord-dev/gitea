// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"fmt"
	"net/http"
	"strings"

	git_model "code.gitea.io/gitea/models/git"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/convert"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/utils"
)

// GetGitAllRefs get ref or an list all the refs of a repository
func GetGitAllRefs(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/git/refs repository repoListAllGitRefs
	// ---
	// summary: Get specified ref or filtered repository's refs
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Reference"
	//     "$ref": "#/responses/ReferenceList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	getGitRefsInternal(ctx, "")
}

// GetGitRefs get ref or an filteresd list of refs of a repository
func GetGitRefs(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/git/refs/{ref} repository repoListGitRefs
	// ---
	// summary: Get specified ref or filtered repository's refs
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: ref
	//   in: path
	//   description: part or full name of the ref
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Reference"
	//     "$ref": "#/responses/ReferenceList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	getGitRefsInternal(ctx, ctx.Params("*"))
}

func getGitRefsInternal(ctx *context.APIContext, filter string) {
	refs, lastMethodName, err := utils.GetGitRefs(ctx, filter)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, lastMethodName, err)
		return
	}

	if len(refs) == 0 {
		ctx.NotFound()
		return
	}

	apiRefs := make([]*api.Reference, len(refs))
	for i := range refs {
		apiRefs[i] = convert.ToGitRef(ctx.Repo.Repository, refs[i])
	}
	// If single reference is found and it matches filter exactly return it as object
	if len(apiRefs) == 1 && apiRefs[0].Ref == filter {
		ctx.JSON(http.StatusOK, &apiRefs[0])
		return
	}
	ctx.JSON(http.StatusOK, &apiRefs)
}

// CreateGitRef creates a git ref for a repository that points to a target commitish
func CreateGitRef(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/git/refs repository repoCreateGitRef
	// ---
	// summary: Create a reference
	// description: Creates a reference for your repository. You are unable to create new references for empty repositories,
	//             even if the commit SHA-1 hash used exists. Empty repositories are repositories without branches.
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateGitRefOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Reference"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     description: The git ref with the same name already exists.
	//   "422":
	//     description: Unable to form reference

	opt := web.GetForm(ctx).(*api.CreateGitRefOption)

	if ctx.Repo.GitRepo.IsReferenceExist(opt.RefName) {
		ctx.Error(http.StatusConflict, "reference already exists:", fmt.Errorf("reference already exists: %s", opt.RefName))
		return
	}

	ref, err := updateReference(ctx, opt.RefName, opt.Target)
	if err != nil {
		return
	}
	ctx.JSON(http.StatusCreated, ref)
}

// UpdateGitRef updates a branch for a repository from a commit SHA
func UpdateGitRef(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/git/refs/{ref} repository repoUpdateGitRef
	// ---
	// summary: Update a reference
	// description:
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: ref
	//   in: path
	//   description: name of the ref to update
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/UpdateGitRefOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Reference"
	//   "404":
	//     "$ref": "#/responses/notFound"

	refName := fmt.Sprintf("refs/%s", ctx.Params("*"))
	opt := web.GetForm(ctx).(*api.UpdateGitRefOption)

	if !ctx.Repo.GitRepo.IsReferenceExist(refName) {
		ctx.Error(http.StatusNotFound, "git ref does not exist:", fmt.Errorf("reference does not exist: %s", refName))
		return
	}

	ref, err := updateReference(ctx, refName, opt.Target)
	if err != nil {
		return
	}
	ctx.JSON(http.StatusOK, ref)
}

// DeleteGitRef deletes a git ref for a repository that points to a target commitish
func DeleteGitRef(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/git/refs/{ref} repository repoDeleteGitRef
	// ---
	// summary: Delete a reference
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: ref
	//   in: path
	//   description: name of the ref to be deleted
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "405":
	//     "$ref": "#/responses/error"
	//   "409":
	//     "$ref": "#/responses/conflict"

	refName := fmt.Sprintf("refs/%s", ctx.Params("*"))

	if !ctx.Repo.GitRepo.IsReferenceExist(refName) {
		ctx.Error(http.StatusNotFound, "git ref does not exist:", fmt.Errorf("reference does not exist: %s", refName))
		return
	}

	_, err := updateReference(ctx, refName, "")
	if err != nil {
		return
	}
	ctx.Status(http.StatusNoContent)
}

// updateReference is used for Create, Update and Deletion of a reference, checking for format, permissions and special cases
func updateReference(ctx *context.APIContext, refName, target string) (*api.Reference, error) {
	if !strings.HasPrefix(refName, "refs/") {
		err := git.ErrInvalidRefName{
			RefName: refName,
			Reason:  "reference must start with 'refs/'",
		}
		ctx.Error(http.StatusUnprocessableEntity, "bad reference'", err)
		return nil, err
	}

	if strings.HasPrefix(refName, "refs/pull/") {
		err := git.ErrInvalidRefName{
			RefName: refName,
			Reason:  "refs/pull/* is read-only",
		}
		ctx.Error(http.StatusUnprocessableEntity, "reference is read-only'", err)
		return nil, err
	}

	if !userCanModifyRef(ctx, refName) {
		err := git.ErrProtectedRefName{
			RefName: refName,
		}
		ctx.Error(http.StatusMethodNotAllowed, "protected ref named", err)
		return nil, err
	}

	// If target is not empty, we update a ref (will create new one if doesn't exist),
	//   else if target is empty, we delete the ref.
	if target != "" {
		commitID, err := ctx.Repo.GitRepo.GetRefCommitID(target)
		if err != nil {
			if git.IsErrNotExist(err) {
				err := fmt.Errorf("target does not exist: %s", target)
				ctx.Error(http.StatusNotFound, "target does not exist", err)
				return nil, err
			}
			ctx.InternalServerError(err)
			return nil, err
		}
		if err := ctx.Repo.GitRepo.SetReference(refName, commitID); err != nil {
			message := err.Error()
			prefix := fmt.Sprintf("exit status 128 - fatal: update_ref failed for ref '%s': ", refName)
			if strings.HasPrefix(message, prefix) {
				message = strings.TrimRight(strings.TrimPrefix(message, prefix), "\n")
				ctx.Error(http.StatusUnprocessableEntity, "reference update failed", message)
			} else {
				ctx.InternalServerError(err)
			}
			return nil, err
		}
		ref, err := ctx.Repo.GitRepo.GetReference(refName)
		if err != nil {
			if git.IsErrRefNotFound(err) {
				ctx.Error(http.StatusUnprocessableEntity, "reference update failed", err)
			} else {
				ctx.InternalServerError(err)
			}
			return nil, err
		}
		return convert.ToGitRef(ctx.Repo.Repository, ref), nil
	} else if err := ctx.Repo.GitRepo.RemoveReference(refName); err != nil {
		ctx.InternalServerError(err)
		return nil, err
	}
	return nil, nil
}

// userCanModifyRef checks based on the reference prefix if the user can modify the reference
func userCanModifyRef(ctx *context.APIContext, ref string) bool {
	refPrefix, refName := git.SplitRefName(ref)
	if refPrefix == "refs/tags/" {
		if protectedTags, err := git_model.GetProtectedTags(ctx.Repo.Repository.ID); err == nil {
			if isAllowed, err := git_model.IsUserAllowedToControlTag(protectedTags, refName, ctx.Doer.ID); err == nil {
				return isAllowed
			}
		}
		return false
	}
	if refPrefix == "refs/heads/" {
		if isProtected, err := git_model.IsProtectedBranch(ctx.Repo.Repository.ID, refName); err == nil {
			return !isProtected
		}
		return false
	}
	return true
}
