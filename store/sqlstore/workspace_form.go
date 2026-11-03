// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/twigex/twigex/model"
)

// A form's tab can be renamed from the tab menu, so its name is read from
// the tab.
const formColumns = `id, workspace_id, table_id,
	COALESCE((SELECT v.name FROM workspace_view v WHERE v.id = workspace_forms.id), workspace_forms.name), title, COALESCE(description, ''), COALESCE(cover_image, ''),
	cover_position, COALESCE(logo, ''), COALESCE(accent_color, ''), public_token, requires_login, active,
	created_by, created_at, updated_at`

// CreateForm writes the form with the workspace_view row that shows it as a
// tab, so a form never exists without its view or the other way round.
func (w *workspaceRepository) CreateForm(ctx context.Context, form model.WorkspaceForm, fields []model.WorkspaceFormField) error {
	tx, err := w.Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_view (id, workspace_id, table_id, view_type, parent_table_id, name, item_order, created_by, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, 'form', '', ?, '', ?, ?, 0, 0)`,
		form.ID, form.WorkspaceID, form.TableID, form.Name, form.CreatedBy, form.CreatedAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_forms
		(id, workspace_id, table_id, name, title, description, cover_image, cover_position, logo, accent_color,
		 public_token, requires_login, active, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		form.ID, form.WorkspaceID, form.TableID, form.Name, form.Title, form.Description,
		form.CoverImage, form.CoverPosition, form.Logo, form.AccentColor,
		form.PublicToken, form.RequiresLogin, form.Active, form.CreatedBy, form.CreatedAt, form.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if err := createFormFieldsTx(ctx, tx, form.ID, fields); err != nil {
		return err
	}

	return tx.Commit()
}

func (w *workspaceRepository) GetForm(ctx context.Context, formID string) (*model.WorkspaceFormWithFields, error) {
	var form model.WorkspaceForm
	err := w.Db.QueryRowContext(ctx, `SELECT `+formColumns+` FROM workspace_forms WHERE id = ?`, formID).Scan(
		&form.ID, &form.WorkspaceID, &form.TableID, &form.Name, &form.Title, &form.Description,
		&form.CoverImage, &form.CoverPosition, &form.Logo, &form.AccentColor, &form.PublicToken,
		&form.RequiresLogin, &form.Active, &form.CreatedBy, &form.CreatedAt, &form.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	fields, err := w.getFormFields(ctx, form.ID, form.TableID)
	if err != nil {
		return nil, err
	}

	return &model.WorkspaceFormWithFields{WorkspaceForm: form, Fields: fields}, nil
}

// UpdateForm replaces the form's fields and keeps its view's name in step.
func (w *workspaceRepository) UpdateForm(ctx context.Context, form model.WorkspaceForm, fields []model.WorkspaceFormField) error {
	tx, err := w.Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE workspace_forms
		SET name = ?, title = ?, description = ?, cover_image = ?, cover_position = ?, logo = ?, accent_color = ?,
		    requires_login = ?, active = ?, updated_at = ?
		WHERE id = ?`,
		form.Name, form.Title, form.Description, form.CoverImage, form.CoverPosition, form.Logo, form.AccentColor,
		form.RequiresLogin, form.Active, form.UpdatedAt, form.ID,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE workspace_view SET name = ?, updated_at = ? WHERE id = ?`, form.Name, form.UpdatedAt, form.ID)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_form_fields WHERE form_id = ?`, form.ID); err != nil {
		return err
	}

	if err := createFormFieldsTx(ctx, tx, form.ID, fields); err != nil {
		return err
	}

	return tx.Commit()
}

func (w *workspaceRepository) DeleteForm(ctx context.Context, formID string) error {
	tx, err := w.Db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_forms WHERE id = ?`, formID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_view WHERE id = ? AND view_type = 'form'`, formID); err != nil {
		return err
	}

	return tx.Commit()
}

func createFormFieldsTx(ctx context.Context, tx *sql.Tx, formID string, fields []model.WorkspaceFormField) error {
	for _, f := range fields {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO workspace_form_fields
			(id, form_id, header_id, header_name, label, placeholder, description, required, visible, sort_order, column_position)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.ID, formID, f.HeaderID, f.HeaderName, f.Label, f.Placeholder, f.Description,
			f.Required, f.Visible, f.SortOrder, f.ColumnPosition,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *workspaceRepository) getFormFields(ctx context.Context, formID, tableID string) ([]model.WorkspaceFormField, error) {
	rows, err := w.Db.QueryContext(ctx, `
		SELECT f.id, f.form_id, f.header_id, COALESCE(NULLIF(f.header_name, ''), wf.field_name, ''), f.label,
		       COALESCE(f.placeholder, ''), COALESCE(f.description, ''), f.required, f.visible, f.sort_order, f.column_position
		FROM workspace_form_fields f
		LEFT JOIN workspace_fields wf ON f.header_name = '' AND wf.id = f.header_id AND wf.deleted_at = 0
		WHERE f.form_id = ?
		ORDER BY f.sort_order`, formID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	fields := make([]model.WorkspaceFormField, 0)
	for rows.Next() {
		var f model.WorkspaceFormField
		err := rows.Scan(&f.ID, &f.FormID, &f.HeaderID, &f.HeaderName, &f.Label,
			&f.Placeholder, &f.Description, &f.Required, &f.Visible, &f.SortOrder, &f.ColumnPosition)
		if err != nil {
			return nil, err
		}

		fields = append(fields, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, f := range fields {
		if f.HeaderName == "" {
			continue
		}

		field, err := w.getField(ctx, tableID, f.HeaderName)
		if err != nil {
			return nil, err
		}

		if field == nil {
			continue
		}

		fields[i].FieldType = field.FieldType
		fields[i].DisplayName = field.DisplayName

		fields[i].LinkedID, fields[i].SingleSelect, fields[i].ParentTableID, err = w.getFieldLink(ctx, tableID, f.HeaderName)
		if err != nil {
			return nil, err
		}
	}

	return fields, nil
}

func (w *workspaceRepository) getField(ctx context.Context, tableID, fieldName string) (*model.WorkspaceFieldData, error) {
	if system := model.SystemFieldData(tableID, fieldName); system != nil {
		return system, nil
	}

	field := model.WorkspaceFieldData{TableID: tableID, Name: fieldName}
	err := w.Db.QueryRowContext(ctx, `
		SELECT field_type, COALESCE(field_display_name, '')
		FROM workspace_fields
		WHERE table_id = ? AND field_name = ? AND deleted_at = 0
		LIMIT 1`, tableID, fieldName,
	).Scan(&field.FieldType, &field.DisplayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &field, nil
}

// getFieldLink returns the table a link field points to, or "" for a field
// that is not a link.
func (w *workspaceRepository) getFieldLink(ctx context.Context, tableID, fieldName string) (linkedID string, singleSelect bool, parentTableID string, err error) {
	err = w.Db.QueryRowContext(ctx, `
		SELECT r.linked_table_id, COALESCE(t.single_select, 0), COALESCE(t.parent_table_id, '')
		FROM workspace_relationships r
		LEFT JOIN workspace_tables t ON t.id = r.linked_table_id
		WHERE r.table_id = ? AND r.table_name = ?
		LIMIT 1`, tableID, fieldName,
	).Scan(&linkedID, &singleSelect, &parentTableID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, "", nil
	}

	return linkedID, singleSelect, parentTableID, err
}
