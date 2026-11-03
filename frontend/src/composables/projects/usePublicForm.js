// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { computed, onUnmounted, ref, toValue } from "vue";
import { t } from "@/i18n/index.js";
import { close, initialize, onProjectChange } from "@/js/websocket";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import { extractErrorMessage } from "@/utils/errors";
import {
    fillableFormFields,
    formSubmitFields,
    missingRequiredFields,
    shownFormFields,
} from "@/utils/projects/forms";

const STATUS_BY_HTTP = {
    // The API's 401 handler already sends the user to the login page and back.
    401: "login",
    403: "forbidden",
    404: "not_found",
};

// Fills in a form through its link, as the public page and the builder's
// preview both do. live keeps the page's form current while it is open.
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

    const fillable = computed(() => fillableFormFields(form.value?.fields));
    const fields = computed(() => shownFormFields(fillable.value, values.value));

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
        try {
            const { data } = await formService.getPublicForm(toValue(token));

            form.value = data;
            configChanged.value = true;
        } catch (error) {
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

    function startOver() {
        values.value = {};
        fieldErrors.value = {};
        submitError.value = "";
        submitted.value = false;
        configChanged.value = false;
    }

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
