// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { computed, onUnmounted, ref, toValue, watch } from "vue";
import { t } from "@/i18n/index.js";
import { close, initialize, onProjectChange } from "@/js/websocket";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import {
    fillableFormFields,
    formSubmitFields,
    missingRequiredFields,
} from "@/utils/projects/forms";

const STATUS_BY_HTTP = {
    // The API's 401 handler already sends the user to the login page and back.
    401: "login",
    403: "forbidden",
    404: "not_found",
};

// live opens the app's websocket and closes it on unmount, so only a page
// outside the app shell, which has no connection of its own, may pass it.
export function usePublicForm(token, { live = false } = {}) {
    const alertStore = useAlertStore();

    const status = ref("loading");
    const loadError = ref("");
    const form = ref(null);
    const values = ref({});
    const honeypot = ref("");
    const fieldErrors = ref({});
    const submitError = ref("");
    const submitting = ref(false);
    const submitted = ref(false);
    const configChanged = ref(false);
    let reloadRequest = 0;

    const fields = computed(() => fillableFormFields(form.value?.fields));

    function failLoad(error) {
        status.value = STATUS_BY_HTTP[error?.response?.status] || "failed";
        loadError.value = extractErrorMessage(error);
    }

    async function load() {
        status.value = "loading";
        try {
            const { data } = await formService.getPublicForm(toValue(token));

            form.value = data;
            status.value = "ready";
        } catch (error) {
            failLoad(error);
        }
    }

    async function reloadChanged() {
        const current = ++reloadRequest;

        try {
            const { data } = await formService.getPublicForm(toValue(token));

            if (current !== reloadRequest) return;

            form.value = data;
            configChanged.value = true;
            clearFilledErrors();
        } catch (error) {
            if (current !== reloadRequest) return;

            if (STATUS_BY_HTTP[error?.response?.status]) failLoad(error);
            else alertStore.showError(extractErrorMessage(error));
        }
    }

    async function submit() {
        submitError.value = "";
        fieldErrors.value = Object.fromEntries(
            missingRequiredFields(fields.value, values.value).map((name) => [
                name,
                t.value("common.error.required_field"),
            ]),
        );
        if (Object.keys(fieldErrors.value).length > 0) return;

        submitting.value = true;
        try {
            await formService.submitPublicForm(
                toValue(token),
                formSubmitFields(fields.value, values.value),
                honeypot.value,
            );
            submitted.value = true;
        } catch (error) {
            submitError.value = extractErrorMessage(error);
        } finally {
            submitting.value = false;
        }
    }

    function clearFilledErrors() {
        if (Object.keys(fieldErrors.value).length === 0) return;

        const missing = new Set(missingRequiredFields(fields.value, values.value));

        fieldErrors.value = Object.fromEntries(
            Object.entries(fieldErrors.value).filter(([name]) => missing.has(name)),
        );
    }

    function startOver() {
        values.value = {};
        fieldErrors.value = {};
        submitError.value = "";
        submitted.value = false;
        configChanged.value = false;
    }

    watch(values, clearFilledErrors, { deep: true });

    if (live) {
        initialize();
        const stopProjectChange = onProjectChange((change) => {
            if (change.type === "FORM_CONFIG_CHANGED" && change.form_id === form.value?.id) {
                reloadChanged();
            }
        });

        onUnmounted(() => {
            stopProjectChange();
            close();
        });
    }

    load();

    return {
        status,
        loadError,
        form,
        fields,
        values,
        honeypot,
        fieldErrors,
        submitError,
        submitting,
        submitted,
        configChanged,
        load,
        submit,
        startOver,
    };
}
