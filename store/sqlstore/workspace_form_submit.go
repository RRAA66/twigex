// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/twigex/twigex/model"
)

func (w *workspaceRepository) GetLinkableTables(ctx context.Context, workspaceID, excludeTableID string) ([]model.FormLinkTable, error) {
	rows, err := w.Db.QueryContext(ctx, `
		SELECT id, name, COALESCE(display_name, '')
		FROM workspace_tables
		WHERE workspace_id = ?
		  AND single_select = 0
		  AND (parent_table_id IS NULL OR parent_table_id = '')
		  AND (deleted_at IS NULL OR deleted_at = 0)
		  AND id != ?
		ORDER BY name`,
		workspaceID, excludeTableID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tables := make([]model.FormLinkTable, 0)
	for rows.Next() {
		var t model.FormLinkTable
		if err := rows.Scan(&t.ID, &t.Name, &t.DisplayName); err != nil {
			return nil, err
		}

		tables = append(tables, t)
	}

	return tables, rows.Err()
}

// CreateFormTask writes a submitted form's row, its links, its place on the
// table's Kanban boards, its files and the submission record in one
// transaction, so a failure leaves no row without the rest. It returns nil
// when the table is gone, model.ErrLinkTargetMissing when a row to link is
// not there, and model.ErrValueRejected when a value does not fit its column.
func (w *workspaceRepository) CreateFormTask(ctx context.Context, workspaceID, tableID string, task model.FormTask, submission model.WorkspaceFormSubmission) (map[string]any, error) {
	// Read committed, as changeTaskLinksTx needs.
	tx, err := w.Db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	var tableName string
	err = tx.QueryRowContext(ctx, `
		SELECT name FROM workspace_tables
		WHERE workspace_id = ? AND id = ? AND (deleted_at IS NULL OR deleted_at = 0)`,
		workspaceID, tableID,
	).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if err := insertFormTaskTx(ctx, tx, tableName, task); err != nil {
		return nil, err
	}

	for _, field := range slices.Sorted(maps.Keys(task.Links)) {
		if _, err := w.changeTaskLinksTx(ctx, tx, workspaceID, tableID, task.ID, field, task.Links[field], nil); err != nil {
			return nil, err
		}
	}

	row, err := w.SelectTaskByIDTx(tx, tableName, task.ID)
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, sql.ErrNoRows
	}

	if err := w.syncTaskKanbanTx(ctx, tx, workspaceID, tableID, task.ID, task.Selects, *row); err != nil {
		return nil, err
	}

	for _, a := range task.Attachments {
		if err := insertAttachmentTx(ctx, tx, a); err != nil {
			return nil, err
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_form_submissions (id, form_id, task_id, submitted_by, submitted_at)
		VALUES (?, ?, ?, ?, ?)`,
		submission.ID, submission.FormID, submission.TaskID, nullableString(submission.SubmittedBy), submission.SubmittedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return *row, nil
}

func insertFormTaskTx(ctx context.Context, tx *sql.Tx, tableName string, task model.FormTask) error {
	table, err := quoteIdent(tableName)
	if err != nil {
		return err
	}

	columns := []string{"id", "created_at", "created_by"}
	args := []any{task.ID, task.CreatedAt, nullableString(task.CreatedBy)}

	for _, column := range slices.Sorted(maps.Keys(task.Values)) {
		quoted, err := quoteIdent(column)
		if err != nil {
			return err
		}

		columns = append(columns, quoted)
		args = append(args, task.Values[column])
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO "+table+" ("+strings.Join(columns, ", ")+") VALUES ("+sqlPlaceholders(len(args))+")", args...)

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && slices.Contains(rejectedValueErrors, mysqlErr.Number) {
		return model.ErrValueRejected
	}

	return err
}

func insertAttachmentTx(ctx context.Context, tx *sql.Tx, a model.WorkspaceAttachment) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspace_attachments
		(id, workspace_id, table_id, field_id, task_id, user_id, name, size, mime_type, width, height, storage_id, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		a.ID, a.WorkspaceID, a.TableID, a.FieldID, a.TaskID, a.UserID, a.Name,
		a.Size, a.MimeType, a.Width, a.Height, nullableString(a.StorageID), a.CreatedAt, a.UpdatedAt,
	)
	return err
}

// rejectedValueErrors are the MySQL errors for a value its column cannot
// hold: out of range, an incorrect date, an incorrect value, too long.
var rejectedValueErrors = []uint16{1264, 1292, 1366, 1406}

func submissionsWhere(formID string, filter model.FormSubmissionsFilter) (string, []any) {
	where := "form_id = ?"
	args := []any{formID}

	if filter.From > 0 {
		where += " AND submitted_at >= ?"
		args = append(args, filter.From)
	}

	if filter.To > 0 {
		where += " AND submitted_at < ?"
		args = append(args, filter.To)
	}

	return where, args
}

func (w *workspaceRepository) CountFormSubmissions(ctx context.Context, formID string, filter model.FormSubmissionsFilter) (int, error) {
	where, args := submissionsWhere(formID, filter)

	var count int
	err := w.Db.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspace_form_submissions WHERE "+where, args...).Scan(&count)

	return count, err
}

func (w *workspaceRepository) GetFormSubmissions(ctx context.Context, formID string, filter model.FormSubmissionsFilter, limit, offset int) ([]model.WorkspaceFormSubmission, error) {
	where, args := submissionsWhere(formID, filter)

	rows, err := w.Db.QueryContext(ctx, `
		SELECT id, form_id, task_id, COALESCE(submitted_by, ''), submitted_at
		FROM workspace_form_submissions
		WHERE `+where+`
		ORDER BY submitted_at DESC, id DESC
		LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	submissions := make([]model.WorkspaceFormSubmission, 0)
	for rows.Next() {
		var s model.WorkspaceFormSubmission
		if err := rows.Scan(&s.ID, &s.FormID, &s.TaskID, &s.SubmittedBy, &s.SubmittedAt); err != nil {
			return nil, err
		}

		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}
