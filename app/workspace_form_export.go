// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/twigex/twigex/model"
	"github.com/twigex/twigex/tlog"
)

type FormSubmissionsExport struct {
	*TaskExport
	form   *model.WorkspaceFormWithFields
	table  taskExportTable
	fields []model.WorkspaceHeaders
	filter model.FormSubmissionsFilter
	user   model.User
	labels []string
}

func (a *App) PrepareFormSubmissionsExport(ctx context.Context, workspaceID, tableID, formID string, filter model.FormSubmissionsFilter, timezone, clock string, labels map[string]string, user model.User) (*FormSubmissionsExport, *model.AppError) {
	dbCtx, cancel := a.dbCtx(ctx)
	defer cancel()

	form, appErr := a.submissionsForm(dbCtx, workspaceID, tableID, formID, user)
	if appErr != nil {
		return nil, appErr
	}

	failed := model.NewAppError("workspace.form_submissions_failed", http.StatusInternalServerError)

	tableName, err := a.Store.Workspace.GetTableName(tableID)
	if err != nil {
		tlog.Errorw("Failed to retrieve the table of a form export", "form_id", formID, "error", err)
		return nil, failed
	}

	colTypes, err := a.Store.Workspace.GetTableColumnTypes(tableName)
	if err != nil {
		tlog.Errorw("Failed to retrieve the columns of a form export", "form_id", formID, "error", err)
		return nil, failed
	}

	headers := a.buildTableHeaders(tableID, colTypes, false, true)
	byName := make(map[string]model.WorkspaceHeaders, len(headers))
	for _, h := range headers {
		byName[h.Name] = h
	}

	// Submissions sent while the file is written would shift the pages it
	// reads, so it ends at the moment it starts.
	if filter.To == 0 {
		filter.To = time.Now().Unix() + 1
	}

	if filter.From >= filter.To {
		return nil, model.NewAppError("workspace.form_submissions_range_reversed", http.StatusBadRequest)
	}

	filter.TableName = tableName
	filter.Access = a.assignedOnlyFilter(user, workspaceID, tableID)

	export := &FormSubmissionsExport{
		TaskExport: a.newTaskExport(user, timezone, clock),
		form:       form,
		table:      taskExportTable{table: model.WorkspaceTable{ID: tableID, Name: tableName}, headers: byName, list: headers},
		filter:     filter,
		user:       user,
		labels:     []string{labelOr(labels, "submitted_by", "Submitted by"), labelOr(labels, "submitted_at", "Submitted at")},
	}

	for _, f := range form.Fields {
		h, ok := byName[f.HeaderName]
		if !f.Visible || !ok {
			continue
		}

		export.fields = append(export.fields, h)
		export.labels = append(export.labels, exportFieldLabel(labels, f))
	}

	return export, nil
}

// exportFieldLabel takes a field's label as the user's page names it, which
// translates a built-in field the form gives no label of its own.
func exportFieldLabel(labels map[string]string, f model.WorkspaceFormField) string {
	return labelOr(labels, "field:"+f.HeaderName, firstNonEmpty(f.Label, f.DisplayName, f.HeaderName))
}

func labelOr(labels map[string]string, key, fallback string) string {
	if label := strings.TrimSpace(labels[key]); label != "" {
		return label
	}

	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

func (e *FormSubmissionsExport) FileName() string {
	return firstNonEmpty(e.form.Name, "Form") + " " + time.Now().In(e.loc).Format("2006-01-02") + ".csv"
}

// A deleted row keeps its submitter and date with its answers left empty, as
// the submissions list shows it.
func (e *FormSubmissionsExport) WriteCSV(ctx context.Context, w io.Writer, flush func()) error {
	if _, err := io.WriteString(w, "\xEF\xBB\xBF"); err != nil {
		return err
	}

	header := make([]string, len(e.labels))
	for i, label := range e.labels {
		header[i] = inertCell(label)
	}

	out := csv.NewWriter(w)
	if err := out.Write(header); err != nil {
		return err
	}

	timeLayout := e.dateLayout + " 15:04"
	if e.dateLayout == "01/02/2006" {
		timeLayout = e.dateLayout + " 3:04 PM"
	}

	now := time.Now()
	record := make([]string, len(e.labels))
	for offset := 0; ; offset += taskExportChunk {
		submissions, err := e.a.Store.Workspace.GetFormSubmissions(ctx, e.form.ID, e.filter, taskExportChunk, offset)
		if err != nil {
			return fmt.Errorf("read submissions: %w", err)
		}

		if len(submissions) == 0 {
			break
		}

		rows, lookups, err := e.readRows(ctx, submissions)
		if err != nil {
			return err
		}

		if err := e.a.nameSubmitters(submissions); err != nil {
			return fmt.Errorf("name submitters: %w", err)
		}

		for _, s := range submissions {
			record[0] = ""
			if info := s.SubmitterInfo; info != nil {
				record[0] = strings.TrimSpace(info.Name + " " + info.Lastname)
			}

			record[1] = time.Unix(s.SubmittedAt, 0).In(e.loc).Format(timeLayout)

			row := rows[s.TaskID]
			var calcRow map[string]any
			for i, h := range e.fields {
				record[2+i] = ""
				switch {
				case row == nil:
				case h.HeaderUsage == "calculations":
					if calcRow == nil {
						calcRow = calculationRow(row)
					}
					record[2+i] = formCalculation(h.Formula, calcRow, e.loc, now)
				default:
					record[2+i] = e.cell(h, row, lookups)
				}
			}

			for i := range record {
				record[i] = inertCell(record[i])
			}

			if err := out.Write(record); err != nil {
				return err
			}
		}

		out.Flush()
		if err := out.Error(); err != nil {
			return err
		}

		if flush != nil {
			flush()
		}

		if len(submissions) < taskExportChunk {
			break
		}
	}

	out.Flush()
	return out.Error()
}

func (e *FormSubmissionsExport) readRows(ctx context.Context, submissions []model.WorkspaceFormSubmission) (map[string]map[string]any, exportLookups, error) {
	taskIDs := make([]string, 0, len(submissions))
	for _, s := range submissions {
		taskIDs = append(taskIDs, s.TaskID)
	}

	rows, err := e.a.Store.Workspace.GetTasksByIDs(ctx, e.table.table.Name, taskIDs, e.filter.Access)
	if err != nil {
		return nil, exportLookups{}, fmt.Errorf("read submitted rows: %w", err)
	}

	e.a.hydrateLinkedAndSingleSelect(ctx, rows, e.table.list)
	e.a.restrictLinkedItems(ctx, rows, e.table.list, e.user, e.form.WorkspaceID)

	lookups, err := e.lookups(ctx, e.table, rows)
	if err != nil {
		return nil, exportLookups{}, fmt.Errorf("look up names: %w", err)
	}

	byID := make(map[string]map[string]any, len(rows))
	for _, row := range rows {
		if id, _ := row["id"].(string); id != "" {
			byID[id] = row
		}
	}

	return byID, lookups, nil
}
