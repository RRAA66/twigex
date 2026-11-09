<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <aside class="w-60 shrink-0 space-y-4 overflow-y-auto border-l border-gray-200 bg-white p-4">
        <h3
            class="border-b border-gray-100 pb-2 text-xs font-semibold uppercase tracking-wide text-gray-500"
        >
            {{ t("projects.form_view.settings.title") }}
        </h3>

        <ToggleSwitch
            :model-value="form.active"
            :label="t('projects.form_view.settings.active')"
            @update:model-value="emit('update', { active: $event })"
        />

        <div>
            <label :for="`${id}-accent`" class="block text-sm font-medium text-gray-900">
                {{ t("projects.form_view.settings.accent_color") }}
            </label>
            <input
                :id="`${id}-accent`"
                :value="form.accent_color"
                type="color"
                class="mt-1 h-8 w-full cursor-pointer rounded border border-gray-200 p-0.5"
                @input="emit('update', { accent_color: $event.target.value })"
            />
        </div>

        <FormViewImageInput
            :label="t('projects.form_view.settings.cover_image')"
            :alt="t('projects.form_view.image_alt.cover')"
            :src="formImageUrl(form.workspace_id, form.cover_image)"
            :loading="busy === 'cover'"
            :disabled="!form.id || busy !== ''"
            @pick="setImage('cover', $event)"
            @remove="removeImage('cover')"
        />

        <FormViewImageInput
            :label="t('projects.form_view.settings.logo')"
            :alt="t('projects.form_view.image_alt.logo')"
            :src="formImageUrl(form.workspace_id, form.logo)"
            contain
            :loading="busy === 'logo'"
            :disabled="!form.id || busy !== ''"
            @pick="setImage('logo', $event)"
            @remove="removeImage('logo')"
        />

        <p v-if="!form.id" class="text-xs text-gray-400">
            {{ t("projects.form_view.settings.images_after_save") }}
        </p>

        <h3
            class="border-b border-gray-100 pb-2 pt-2 text-xs font-semibold uppercase tracking-wide text-gray-500"
        >
            {{ t("projects.form_view.settings.after_submit") }}
        </h3>

        <div>
            <label :for="`${id}-thanks`" class="block text-sm font-medium text-gray-900">
                {{ t("projects.form_view.settings.success_message") }}
            </label>
            <textarea
                :id="`${id}-thanks`"
                :value="form.success_message"
                rows="3"
                :placeholder="t('projects.form_view.success.body')"
                class="mt-1 block w-full rounded-md border-0 py-1.5 text-sm text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600"
                @input="emit('update', { success_message: $event.target.value })"
            />
        </div>

        <BaseSelect
            :model-value="form.after_submit"
            :label="t('projects.form_view.settings.after_submit_then')"
            :options="afterSubmitOptions"
            @update:model-value="chooseAfterSubmit"
        />

        <div v-if="form.after_submit === 'link'">
            <label :for="`${id}-redirect`" class="block text-sm font-medium text-gray-900">
                {{ t("projects.form_view.settings.redirect_url") }}
            </label>
            <input
                :id="`${id}-redirect`"
                :value="form.redirect_url"
                type="url"
                placeholder="https://"
                :aria-describedby="`${id}-redirect-hint`"
                class="mt-1 block w-full rounded-md border-0 py-1.5 text-sm text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600"
                @input="emit('update', { redirect_url: $event.target.value.trim() })"
            />
            <p :id="`${id}-redirect-hint`" class="mt-1 text-xs text-gray-500">
                {{ t("projects.form_view.settings.redirect_hint") }}
            </p>
        </div>
    </aside>
</template>

<script setup>
import { computed, useId } from "vue";
import { t } from "@/i18n/index.js";
import { useFormImages } from "@/composables/projects/useFormImages";
import { formImageUrl } from "@/utils/projects/forms";
import BaseSelect from "@/components/BaseSelect.vue";
import ToggleSwitch from "@/components/ToggleSwitch.vue";
import FormViewImageInput from "@/components/Projects/Form/FormViewImageInput.vue";

const props = defineProps({
    form: {
        type: Object,
        required: true,
    },
});

const emit = defineEmits(["update"]);

const { busy, setImage, removeImage } = useFormImages(
    () => props.form,
    (patch) => emit("update", patch),
);

const id = useId();

const afterSubmitOptions = computed(() => [
    { value: "", label: t.value("projects.form_view.settings.after_submit_stay") },
    { value: "grid", label: t.value("projects.form_view.settings.after_submit_grid") },
    { value: "link", label: t.value("projects.form_view.settings.after_submit_link") },
]);

// A link is kept only while it is where the form goes.
function chooseAfterSubmit(afterSubmit) {
    emit("update", {
        after_submit: afterSubmit,
        redirect_url: afterSubmit === "link" ? props.form.redirect_url : "",
    });
}
</script>
