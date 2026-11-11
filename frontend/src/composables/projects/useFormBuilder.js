// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { computed, ref, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t } from "@/i18n/index.js";
import formService from "@/services/formService";
import { useFormImages } from "@/composables/projects/useFormImages";
import { useAlertStore } from "@/store/alerts";
import { useSettingsStore } from "@/store/settings";
import { useWorkspaceStore } from "@/store/workspaces";
import { extractErrorMessage } from "@/utils/errors";
import {
    emptyFormData,
    formLinkTables,
    incompleteConditionFields,
    normalizeFormData,
    mergeFormFields,
} from "@/utils/projects/forms";
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
    const conditionsChecked = ref(false);
    const images = useFormImages(formData);

    // Flagged only once a save was tried, so a condition being set up is not
    // marked as wrong.
    const incompleteFields = computed(() =>
        conditionsChecked.value ? incompleteConditionFields(formFields.value) : [],
    );

    let loadRequest = 0;

    async function loadTable() {
        const { data } = await workspaceStore.fetchWorkspaceNav(route.params.id);
        const tables = workspaceTables(data);
        const table = tables.find((tbl) => tbl.id === route.params.tid);

        return {
            headers: table?.headers || [],
            linkTables: formLinkTables(tables, route.params.tid),
        };
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
        conditionsChecked.value = false;
        images.discard();

        if (!settingsStore.getLicenseFeature("forms")) {
            status.value = "unlicensed";

            return;
        }

        status.value = "loading";
        try {
            const table = await loadTable();
            const saved = await loadSavedFields();

            if (request !== loadRequest) return;

            headers.value = table.headers;
            // A saved form brings the server's list; a new one starts from the
            // same tables the grid offers.
            formData.value = saved.form
                ? normalizeFormData(saved.form, id, tid)
                : { ...formData.value, link_tables: table.linkTables };
            formFields.value = mergeFormFields(table.headers, saved.fields, t.value);
            status.value = "ready";
        } catch (error) {
            if (request !== loadRequest) return;

            if (error?.response?.status === 403) {
                status.value = "forbidden";

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

    function checkConditions() {
        conditionsChecked.value = true;

        if (incompleteFields.value.length === 0) return true;

        alertStore.showError(t.value("projects.form_view.error.incomplete_conditions"));

        return false;
    }

    function payloadFields() {
        return formFields.value.map((f, i) => ({ ...f, sort_order: i }));
    }

    function renameViewTab(id, name) {
        workspaceStore.setViewTypes(
            (workspaceStore.getViewTypes || []).map((v) => (v.id === id ? { ...v, name } : v)),
        );
    }

    // The form is already saved when this runs, so a failure says only the
    // images were not.
    async function saveImages() {
        try {
            await images.save(formData.value, (stored) => updateSettings(stored));

            return true;
        } catch (error) {
            alertStore.showError(
                t.value("projects.form_view.error.images_not_saved", {
                    reason: extractErrorMessage(error),
                }),
            );

            return false;
        }
    }

    async function updateForm() {
        if (!checkConditions()) return;

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
            if (request !== loadRequest) {
                alertStore.showSuccess(t.value("projects.form_view.success.saved"));

                return;
            }

            formData.value = normalizeFormData(data, route.params.id, route.params.tid);
            if (await saveImages())
                alertStore.showSuccess(t.value("projects.form_view.success.saved"));
        } catch (error) {
            alertStore.showError(extractErrorMessage(error));
        } finally {
            saving.value = false;
        }
    }

    async function createForm(name) {
        if (!checkConditions()) return false;

        const request = loadRequest;

        saving.value = true;
        try {
            const { data } = await formService.createForm(
                route.params.id,
                route.params.tid,
                { ...formData.value, id: "", name },
                payloadFields(),
            );

            if (request !== loadRequest) {
                alertStore.showSuccess(t.value("projects.form_view.success.saved"));

                return true;
            }

            formData.value = normalizeFormData(data, route.params.id, route.params.tid);
            if (await saveImages())
                alertStore.showSuccess(t.value("projects.form_view.success.saved"));
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
        incompleteFields,
        load,
        checkConditions,
        updateSettings,
        updateField,
        reorderFields,
        updateForm,
        createForm,
        imageSrc: images.src,
        pickImage: images.pick,
        removeImage: images.remove,
    };
}
