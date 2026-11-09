// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import axios from "./api";

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
};

export default formService;
