// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repo

import (
	"fmt"
	"net/http"
	"net/url"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/context"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/utils"
	repo_service "code.gitea.io/gitea/services/repository"
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
		apiRefs[i] = &api.Reference{
			Ref: refs[i].Name,
			URL: ctx.Repo.Repository.APIURL() + "/git/" + util.PathEscapeSegments(refs[i].Name),
			Object: &api.GitObject{
				SHA:  refs[i].Object.String(),
				Type: refs[i].Type,
				URL:  ctx.Repo.Repository.APIURL() + "/git/" + url.PathEscape(refs[i].Type) + "s/" + url.PathEscape(refs[i].Object.String()),
			},
		}
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
	// summary: Create a Git Ref
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
	//     description: The target does not exist.
	//   "409":
	//     description: The git ref with the same name already exists.

	opt := web.GetForm(ctx).(*api.CreateGitRefOption)

	commit, err := ctx.Repo.GitRepo.GetCommit(opt.Target)
	if err != nil {
		ctx.Error(http.StatusNotFound, "target not found", fmt.Errorf("target not found: %v", err))
		return
	}

	if err := repo_service.CreateNewRef(ctx, ctx.Doer, commit.ID.String(), opt.Ref); err != nil {
		if models.IsErrRefAlreadyExists(err) {
			ctx.Error(http.StatusConflict, "ref name exist", err)
			return
		} else if models.IsErrProtectedRefName(err) {
			ctx.Error(http.StatusMethodNotAllowed, "CreateGitRef", "user not allowed to create protected tag")
			return
		}

		ctx.InternalServerError(err)
		return
	}
	retStruct := &api.Reference{
		Ref: opt.Ref,
		URL: ctx.Repo.Repository.APIURL() + "/git/" + util.PathEscapeSegments(opt.Ref),
		Object: &api.GitObject{
			SHA:  commit.ID.String(),
			Type: "unknown-type",
			URL:  ctx.Repo.Repository.APIURL() + "/git/" + url.PathEscape("unknown-type") + "s/" + url.PathEscape(commit.ID.String()),
		},
	}
	ctx.JSON(http.StatusOK, retStruct)
}

// UpdateGitRef updates a branch for a repository from a commit SHA
func UpdateGitRef(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/git/refs repository repoUpdateGitRef
	// ---
	// summary: Update a Git Ref
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
	//     "$ref": "#/definitions/UpdateGitRefOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Reference"
	//   "404":
	//     description: The target or git ref does not exist.

	opt := web.GetForm(ctx).(*api.CreateGitRefOption)

	commit, err := ctx.Repo.GitRepo.GetCommit(opt.Target)
	if err != nil {
		ctx.Error(http.StatusNotFound, "target not found", fmt.Errorf("target not found: %v", err))
		return
	}

	if err := repo_service.CreateNewRef(ctx, ctx.Doer, commit.ID.String(), opt.Ref); err != nil {
		if models.IsErrRefAlreadyExists(err) {
			ctx.Error(http.StatusConflict, "ref name exist", err)
			return
		} else if models.IsErrProtectedRefName(err) {
			ctx.Error(http.StatusMethodNotAllowed, "CreateGitRef", "user not allowed to create protected tag")
			return
		}

		ctx.InternalServerError(err)
		return
	}
	retStruct := &api.Reference{
		Ref: opt.Ref,
		URL: ctx.Repo.Repository.APIURL() + "/git/" + util.PathEscapeSegments(opt.Ref),
		Object: &api.GitObject{
			SHA:  commit.ID.String(),
			Type: "unknown-type",
			URL:  ctx.Repo.Repository.APIURL() + "/git/" + url.PathEscape("unknown-type") + "s/" + url.PathEscape(commit.ID.String()),
		},
	}
	ctx.JSON(http.StatusOK, retStruct)
}

// DeleteGitRef deletes a git ref for a repository that points to a target commitish
func DeleteGifRef(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/git/refs/{ref} repository repoCreateGitRef
	// ---
	// summary: Delete a Git Ref
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
	//   description: ref to be deleted
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	opt := web.GetForm(ctx).(*api.CreateGitRefOption)

	commit, err := ctx.Repo.GitRepo.GetCommit(opt.Target)
	if err != nil {
		ctx.Error(http.StatusNotFound, "target not found", fmt.Errorf("target not found: %v", err))
		return
	}

	if err := repo_service.CreateNewRef(ctx, ctx.Doer, commit.ID.String(), opt.Ref); err != nil {
		if models.IsErrRefAlreadyExists(err) {
			ctx.Error(http.StatusConflict, "ref name exist", err)
			return
		} else if models.IsErrProtectedRefName(err) {
			ctx.Error(http.StatusMethodNotAllowed, "CreateGitRef", "user not allowed to create protected tag")
			return
		}

		ctx.InternalServerError(err)
		return
	}
	retStruct := &api.Reference{
		Ref: opt.Ref,
		URL: ctx.Repo.Repository.APIURL() + "/git/" + util.PathEscapeSegments(opt.Ref),
		Object: &api.GitObject{
			SHA:  commit.ID.String(),
			Type: "unknown-type",
			URL:  ctx.Repo.Repository.APIURL() + "/git/" + url.PathEscape("unknown-type") + "s/" + url.PathEscape(commit.ID.String()),
		},
	}
	ctx.JSON(http.StatusOK, retStruct)
}
