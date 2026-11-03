<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div v-if="status === 'loading' || status === 'login'" class="flex justify-center py-20">
        <BaseSpinner size="lg" class="text-indigo-600" :label="t('common.label.loading')" />
    </div>

    <div
        v-else-if="status !== 'ready'"
        class="flex flex-col items-center gap-y-3 py-20 text-center"
    >
        <component :is="statusIcon" class="h-8 w-8 text-gray-400" aria-hidden="true" />
        <p class="max-w-sm text-sm text-gray-500">{{ statusMessage }}</p>
        <BaseButton v-if="status === 'failed'" size="small" variant="secondary" @click="load">
            {{ t("common.button.try_again") }}
        </BaseButton>
    </div>

    <div v-else-if="submitted" class="flex flex-col items-center gap-y-2 py-20 text-center">
        <CheckCircleIcon class="h-12 w-12 text-green-500" aria-hidden="true" />
        <h2 class="text-xl font-bold text-gray-900">{{ t("projects.form_view.success.title") }}</h2>
        <p class="text-sm text-gray-500">{{ t("projects.form_view.success.body") }}</p>
        <BaseButton class="mt-2" size="small" variant="secondary" @click="startOver">
            {{ t("projects.form_view.action.submit_another") }}
        </BaseButton>
    </div>

    <form
        v-else
        class="w-full overflow-hidden rounded-lg border-t-4 bg-white shadow-sm ring-1 ring-gray-200"
        :style="{ borderTopColor: form.accent_color || DEFAULT_ACCENT_COLOR }"
        novalidate
        @submit.prevent="submit"
    >
        <FormCoverImage
            v-if="form.cover_image"
            :src="formImageUrl(form.workspace_id, form.cover_image)"
            :position="form.cover_position"
        />
        <div class="space-y-2 px-6 pt-8 sm:px-10">
            <p
                v-if="configChanged"
                role="status"
                class="mb-4 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800"
            >
                {{ t("projects.form_view.config_changed") }}
            </p>
            <p
                v-if="!form.can_submit"
                role="status"
                class="mb-4 rounded-md border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-700"
            >
                {{ t("projects.form_view.public.read_only") }}
            </p>
            <img
                v-if="form.logo"
                :src="formImageUrl(form.workspace_id, form.logo)"
                :alt="t('projects.form_view.image_alt.logo')"
                class="h-12 w-auto max-w-48 object-contain"
            />
            <h1 class="text-2xl font-bold text-gray-900">
                {{ form.title || form.name || t("projects.form_view.untitled_form") }}
            </h1>
            <p v-if="form.description" class="whitespace-pre-line text-sm text-gray-500">
                {{ form.description }}
            </p>
        </div>

        <div class="relative px-6 pb-8 pt-6 sm:px-10">
            <div class="absolute -left-[9999px] h-px w-px overflow-hidden" aria-hidden="true">
                <label :for="honeypotId">{{ t("projects.form_view.honeypot_label") }}</label>
                <input
                    :id="honeypotId"
                    v-model="honeypot"
                    type="text"
                    tabindex="-1"
                    autocomplete="off"
                />
            </div>

            <fieldset :disabled="!form.can_submit" class="grid grid-cols-2 gap-4">
                <FormFieldInput
                    v-for="field in fields"
                    :key="field.header_name"
                    v-model="values[field.header_name]"
                    :field="field"
                    :token="token"
                    :members="form.members"
                    :link-tables="form.link_tables"
                    :error="fieldErrors[field.header_name]"
                />
            </fieldset>

            <p v-if="submitError" role="alert" class="mt-4 text-sm text-red-600">
                {{ submitError }}
            </p>

            <BaseButton
                v-if="form.can_submit"
                type="submit"
                class="mt-6 w-full"
                color="text-white hover:opacity-90"
                :style="{ backgroundColor: form.accent_color || DEFAULT_ACCENT_COLOR }"
                :is-loading="submitting"
            >
                {{ t("projects.form_view.action.submit") }}
            </BaseButton>
        </div>
    </form>
</template>

<script setup>
import { computed, useId } from "vue";
import {
    CheckCircleIcon,
    DocumentMagnifyingGlassIcon,
    ExclamationTriangleIcon,
    LockClosedIcon,
} from "@heroicons/vue/24/outline";
import { t } from "@/i18n/index.js";
import { usePublicForm } from "@/composables/projects/usePublicForm";
import { DEFAULT_ACCENT_COLOR, formImageUrl } from "@/utils/projects/forms";
import BaseButton from "@/components/BaseButton.vue";
import BaseSpinner from "@/components/BaseSpinner.vue";
import FormCoverImage from "@/components/Projects/Form/FormCoverImage.vue";
import FormFieldInput from "@/components/Projects/Form/FormFieldInput.vue";

const props = defineProps({
    token: { type: String, required: true },
    live: { type: Boolean, default: false },
});

const {
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
} = usePublicForm(() => props.token, { live: props.live });

const honeypotId = useId();

const statusIcon = computed(
    () =>
        ({ forbidden: LockClosedIcon, not_found: DocumentMagnifyingGlassIcon })[status.value] ||
        ExclamationTriangleIcon,
);

const statusMessage = computed(() => {
    if (status.value === "forbidden") return t.value("projects.form_view.public.no_access");
    if (status.value === "not_found") return t.value("projects.form_view.public.not_found");

    return loadError.value;
});
</script>
