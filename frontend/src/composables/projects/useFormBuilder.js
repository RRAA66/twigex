// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { ref, onMounted, watch } from "vue";
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

    async function loadHeaders() {
        const { data } = await workspaceStore.fetchWorkspaceNav(route.params.id);
        const table = workspaceTables(data).find((tbl) => tbl.id === route.params.tid);

        headers.value = table?.headers || [];
    }

    async function loadForm(formId) {
        const { data } = await formService.getForm(route.params.id, route.params.tid, formId);

        formData.value = normalizeFormData(data, route.params.id, route.params.tid);

        return data.fields || [];
    }

    async function load() {
        formData.value = emptyFormData(route.params.id, route.params.tid);
        formFields.value = [];

        if (!settingsStore.getLicenseFeature("forms")) {
            status.value = "unlicensed";

            return;
        }

        status.value = "loading";
        try {
            await loadHeaders();

            const savedFields = route.params.fid ? await loadForm(route.params.fid) : [];

            formFields.value = mergeFormFields(headers.value, savedFields, t.value);
            status.value = "ready";
        } catch (error) {
            if (error?.response?.status === 403) {
                status.value = "forbidden";

                return;
            }

            status.value = "failed";
            alertStore.showError(extractErrorMessage(error));
        }
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
        saving.value = true;
        try {
            const { data } = await formService.updateForm(
                route.params.id,
                route.params.tid,
                formData.value.id,
                formData.value,
                payloadFields(),
            );

            formData.value = normalizeFormData(data, route.params.id, route.params.tid);
            renameViewTab(data.id, data.name);
            alertStore.showSuccess(t.value("projects.form_view.success.saved"));
        } catch (error) {
            alertStore.showError(extractErrorMessage(error));
        } finally {
            saving.value = false;
        }
    }

    async function createForm(name) {
        saving.value = true;
        try {
            const { data } = await formService.createForm(
                route.params.id,
                route.params.tid,
                { ...formData.value, id: "", name },
                payloadFields(),
            );

            formData.value = normalizeFormData(data, route.params.id, route.params.tid);
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
        () => route.params.fid,
        (fid) => {
            if ((fid || "") !== formData.value.id) load();
        },
    );

    return {
        formData,
        formFields,
        headers,
        status,
        saving,
        load,
        updateForm,
        createForm,
    };
}
