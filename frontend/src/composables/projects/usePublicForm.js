// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

import { computed, onUnmounted, ref, toValue, watch } from "vue";
import { t } from "@/i18n/index.js";
import { close, initialize, onProjectChange } from "@/js/websocket";
import formService from "@/services/formService";
import { useAlertStore } from "@/store/alerts";
import {
    defaultAnswers,
    fillableFormFields,
    formRequestError,
    formSubmitFields,
    invalidNumberFields,
    missingRequiredFields,
    shownFormFields,
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

    const fillable = computed(() => fillableFormFields(form.value?.fields));
    const fields = computed(() => shownFormFields(fillable.value, values.value));

    function failLoad(error) {
        status.value = STATUS_BY_HTTP[error?.response?.status] || "failed";
        loadError.value = formRequestError(error, t.value);
    }

    async function load() {
        status.value = "loading";
        try {
            const { data } = await formService.getPublicForm(toValue(token));

            form.value = data;
            values.value = defaultAnswers(fillableFormFields(data.fields));
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
            else alertStore.showError(formRequestError(error, t.value));
        }
    }

    async function submit() {
        submitError.value = "";

        const numberErrors = Object.entries(invalidNumberFields(fields.value, values.value)).map(
            ([name, key]) => [name, t.value(key)],
        );
        const requiredErrors = missingRequiredFields(fields.value, values.value).map((name) => [
            name,
            t.value("common.error.required_field"),
        ]);

        fieldErrors.value = Object.fromEntries([...numberErrors, ...requiredErrors]);
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
            submitError.value = formRequestError(error, t.value);
        } finally {
            submitting.value = false;
        }
    }

    function clearFilledErrors() {
        if (Object.keys(fieldErrors.value).length === 0) return;

        const missing = new Set(missingRequiredFields(fields.value, values.value));
        const invalid = invalidNumberFields(fields.value, values.value);

        fieldErrors.value = Object.fromEntries(
            Object.entries(fieldErrors.value).filter(
                ([name]) => missing.has(name) || invalid[name],
            ),
        );
    }

    function startOver() {
        values.value = defaultAnswers(fillable.value);
        fieldErrors.value = {};
        submitError.value = "";
        submitted.value = false;
        configChanged.value = false;
    }

    watch(
        values,
        () => {
            submitError.value = "";
            clearFilledErrors();
        },
        { deep: true },
    );

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
