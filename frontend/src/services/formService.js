// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import axios from "./api";

const MULTIPART = { headers: { "Content-Type": "multipart/form-data" } };

const formService = {
    createForm(workspaceId, tableId, form, fields) {
        return axios.post(`/workspaces/${workspaceId}/table/${tableId}/forms`, { form, fields });
    },

    getForm(workspaceId, tableId, formId) {
        return axios.get(`/workspaces/${workspaceId}/table/${tableId}/forms/${formId}`);
    },

    updateForm(workspaceId, tableId, formId, form, fields) {
        return axios.put(`/workspaces/${workspaceId}/table/${tableId}/forms/${formId}`, {
            form,
            fields,
        });
    },

    deleteForm(workspaceId, tableId, formId) {
        return axios.delete(`/workspaces/${workspaceId}/table/${tableId}/forms/${formId}`);
    },

    getSubmissions(workspaceId, tableId, formId) {
        return axios.get(`/workspaces/${workspaceId}/table/${tableId}/forms/${formId}/submissions`);
    },

    getPublicForm(token) {
        return axios.get(`/public/forms/${token}`);
    },

    getPublicLinkedItems(token, tableId, params) {
        return axios.get(`/public/forms/${token}/linked/${tableId}`, { params });
    },

    submitPublicForm(token, fields, honeypot) {
        return axios.post(`/public/forms/${token}/submit`, { fields, honeypot });
    },

    uploadPublicFormFiles(token, files) {
        const data = new FormData();

        for (const file of files) data.append("files", file);

        return axios.post(`/public/forms/${token}/files`, data, MULTIPART);
    },

    setFormImage(workspaceId, tableId, formId, kind, file) {
        const data = new FormData();

        data.append("file", file);

        return axios.post(
            `/workspaces/${workspaceId}/table/${tableId}/forms/${formId}/images/${kind}`,
            data,
            MULTIPART,
        );
    },

    removeFormImage(workspaceId, tableId, formId, kind) {
        return axios.delete(
            `/workspaces/${workspaceId}/table/${tableId}/forms/${formId}/images/${kind}`,
        );
    },
};

export default formService;
