// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { computed, ref, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t } from "@/i18n/index.js";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { useSettingsStore } from "@/store/settings";
import { useWorkspaceStore } from "@/store/workspaces";
import { extractErrorMessage } from "@/utils/errors";
import { emptyFormData, normalizeFormData, mergeFormFields } from "@/utils/projects/forms";
import { workspaceTables } from "@/utils/projects/tree";

export function useFormBuilder() {
    const route = useRoute();
    const router = useRouter();
    const alertStore = useAlertStore();
    const settingsStore = useSettingsStore();
    const workspaceStore = useWorkspaceStore();

    const formData = ref(emptyFormData(route.params.id, route.params.tid));
    const formFields = ref([]);
    const headers = ref([]);
    const status = ref("loading");
    const saving = ref(false);
    const changedElsewhere = ref(false);
    const savedState = ref("");

    let loadRequest = 0;

    const isDirty = computed(() => status.value === "ready" && currentState() !== savedState.value);

    function currentState() {
        return JSON.stringify([formData.value, formFields.value]);
    }

    function markSaved() {
        savedState.value = currentState();
    }

    async function loadHeaders() {
        const { data } = await workspaceStore.fetchWorkspaceNav(route.params.id);
        const table = workspaceTables(data).find((tbl) => tbl.id === route.params.tid);

        return table?.headers || [];
    }

    async function loadSavedFields() {
        if (!route.params.fid) return { form: null, fields: [] };

        const { data } = await formService.getForm(
            route.params.id,
            route.params.tid,
            route.params.fid,
        );

        return { form: data, fields: data.fields || [] };
    }

    async function load() {
        const request = ++loadRequest;
        const { id, tid } = route.params;

        formData.value = emptyFormData(id, tid);
        formFields.value = [];
        changedElsewhere.value = false;

        if (!settingsStore.getLicenseFeature("forms")) {
            status.value = "unlicensed";

            return;
        }

        status.value = "loading";
        try {
            const tableHeaders = await loadHeaders();
            const saved = await loadSavedFields();

            if (request !== loadRequest) return;

            headers.value = tableHeaders;
            if (saved.form) formData.value = normalizeFormData(saved.form, id, tid);
            formFields.value = mergeFormFields(tableHeaders, saved.fields);
            markSaved();
            status.value = "ready";
        } catch (error) {
            if (request !== loadRequest) return;

            if (error?.response?.status === 403) {
                status.value = "forbidden";

                return;
            }

            if (error?.response?.status === 404) {
                status.value = "missing";

                return;
            }

            status.value = "failed";
            alertStore.showError(extractErrorMessage(error));
        }
    }

    function updateSettings(patch) {
        formData.value = { ...formData.value, ...patch };
    }

    function updateField(headerName, patch) {
        formFields.value = formFields.value.map((f) =>
            f.header_name === headerName ? { ...f, ...patch } : f,
        );
    }

    function reorderFields(fields) {
        formFields.value = fields;
    }

    function payloadFields() {
        return formFields.value.map((f, i) => ({ ...f, sort_order: i }));
    }

    function renameViewTab(id, name) {
        workspaceStore.setViewTypes(
            (workspaceStore.getViewTypes || []).map((v) => (v.id === id ? { ...v, name } : v)),
        );
    }

    async function updateForm() {
        const request = loadRequest;

        saving.value = true;
        try {
            const { data } = await formService.updateForm(
                route.params.id,
                route.params.tid,
                formData.value.id,
                formData.value,
                payloadFields(),
            );

            renameViewTab(data.id, data.name);
            if (request === loadRequest) {
                formData.value = normalizeFormData(data, route.params.id, route.params.tid);
                markSaved();
            }

            alertStore.showSuccess(t.value("projects.form_view.success.saved"));
        } catch (error) {
            if (error?.response?.status === 409 && request === loadRequest) {
                changedElsewhere.value = true;

                return;
            }

            alertStore.showError(extractErrorMessage(error));
        } finally {
            saving.value = false;
        }
    }

    async function createForm(name) {
        const request = loadRequest;

        saving.value = true;
        try {
            const { data } = await formService.createForm(
                route.params.id,
                route.params.tid,
                { ...formData.value, id: "", name },
                payloadFields(),
            );

            alertStore.showSuccess(t.value("projects.form_view.success.saved"));
            if (request !== loadRequest) return true;

            formData.value = normalizeFormData(data, route.params.id, route.params.tid);
            markSaved();
            await router.replace({
                name: "form-view",
                params: { id: route.params.id, tid: route.params.tid, fid: data.id },
            });

            return true;
        } catch (error) {
            alertStore.showError(extractErrorMessage(error));

            return false;
        } finally {
            saving.value = false;
        }
    }

    onMounted(load);

    watch(
        () => [route.params.id, route.params.tid, route.params.fid],
        ([id, tid, fid]) => {
            const form = formData.value;

            if (id !== form.workspace_id || tid !== form.table_id || (fid || "") !== form.id) {
                load();
            }
        },
    );

    return {
        formData,
        formFields,
        headers,
        status,
        saving,
        changedElsewhere,
        isDirty,
        load,
        updateSettings,
        updateField,
        reorderFields,
        updateForm,
        createForm,
    };
}
