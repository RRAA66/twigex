// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

// checkFormDefaults refuses a default its field would refuse as an answer,
// and one on a file field, which has nothing to start from.
func (a *App) checkFormDefaults(ctx context.Context, workspaceID, tableID string, fields []model.WorkspaceFormField, user model.User, failedID string) *model.AppError {
	invalid := model.NewAppError("request.invalid", http.StatusBadRequest)

	withDefault := slices.DeleteFunc(slices.Clone(fields), func(f model.WorkspaceFormField) bool {
		return f.DefaultValue == ""
	})
	if len(withDefault) == 0 {
		return nil
	}

	meta, err := a.Store.Workspace.GetTableHeaderMeta(tableID)
	if err != nil {
		tlog.Errorw("Failed to load the form's table fields", "table_id", tableID, "error", err)
		return model.NewAppError(failedID, http.StatusInternalServerError)
	}

	for i := range withDefault {
		if !meta.DescribeFormField(tableID, &withDefault[i]) {
			return invalid
		}
	}

	linkTables := map[string]bool{}
	if slices.ContainsFunc(withDefault, isMasterLinkField) {
		form := &model.WorkspaceFormWithFields{WorkspaceForm: model.WorkspaceForm{WorkspaceID: workspaceID, TableID: tableID}}

		tables, appErr := a.formLinkTables(ctx, user, form)
		if appErr != nil {
			return appErr
		}

		for _, t := range tables {
			linkTables[t.ID] = true
		}
	}

	scratch := model.FormTask{Values: map[string]any{}, Links: map[string][]string{}, Selects: map[string]string{}}
	for _, f := range withDefault {
		if f.FieldType == "file" || isCalculationField(f) || len(f.DefaultValue) > maxFormTextBytes {
			return invalid
		}

		if appErr := a.setFormValue(ctx, workspaceID, f, f.DefaultValue, linkTables, &scratch); appErr != nil {
			return appErr
		}
	}

	return nil
}

// presetFormValues answers the defaults of the fields the form does not
// show, which no one can change.
func (a *App) presetFormValues(ctx context.Context, form *model.WorkspaceFormWithFields, linkTables map[string]bool, task *model.FormTask) *model.AppError {
	for _, f := range form.Fields {
		if f.Visible || f.DefaultValue == "" || f.FieldType == "file" || isCalculationField(f) || model.IsServerKeptColumn(f.HeaderName) {
			continue
		}

		if appErr := a.setFormValue(ctx, form.WorkspaceID, f, f.DefaultValue, linkTables, task); appErr != nil {
			return appErr
		}
	}

	return nil
}

// prepareBuilderForm adds what the builder shows beside the saved form: the
// names of linked defaults and the tables a master link may name.
func (a *App) prepareBuilderForm(ctx context.Context, user model.User, form *model.WorkspaceFormWithFields) *model.AppError {
	a.nameFormDefaults(ctx, user, form.WorkspaceID, form.Fields)

	tables, appErr := a.formLinkTables(ctx, user, form)
	if appErr != nil {
		return appErr
	}
	form.LinkTables = tables
	return nil
}

// nameFormDefaults gives a linked field's default the names its picker
// shows, from the rows the user may see.
func (a *App) nameFormDefaults(ctx context.Context, user model.User, workspaceID string, fields []model.WorkspaceFormField) {
	for i, f := range fields {
		if f.DefaultValue == "" || f.LinkedID == "" {
			continue
		}

		tableID := pickedTable(f)

		access := model.SQLFilter{}
		if !f.SingleSelect {
			if !a.canReadTable(ctx, user, workspaceID, tableID) {
				continue
			}

			access = a.assignedOnlyFilter(user, workspaceID, tableID)
		}

		ids := strings.Split(f.DefaultValue, ",")

		tableName, err := a.Store.Workspace.GetTableName(tableID)
		var rows []map[string]any
		if err == nil {
			rows, err = a.Store.Workspace.GetLinkedRecordsLite(ctx, tableName, "", ids, len(ids), 0, access)
		}

		if err != nil {
			tlog.Warnw("Failed to name a form field's default", "table_id", tableID, "error", err)
			continue
		}

		byID := make(map[string]model.FormLinkedItem, len(rows))
		for _, row := range rows {
			id, _ := row["id"].(string)
			name, _ := row["name"].(string)
			color, _ := row["color"].(string)
			byID[id] = model.FormLinkedItem{ID: id, Name: name, Color: color}
		}

		for _, id := range ids {
			if item, ok := byID[id]; ok {
				fields[i].DefaultItems = append(fields[i].DefaultItems, item)
			}
		}
	}
}
