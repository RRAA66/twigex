<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="flex min-h-0 flex-1">
        <div class="flex-1 space-y-3 overflow-y-auto p-6">
            <div
                class="overflow-hidden rounded-lg border border-t-4 border-gray-200 bg-white shadow-sm"
                :style="{ borderTopColor: form.accent_color }"
            >
                <FormCoverImage
                    v-if="coverSrc"
                    :position="form.cover_position"
                    :src="coverSrc"
                    repositionable
                    @update:position="emit('change-settings', { cover_position: $event })"
                />
                <div class="space-y-2 p-6">
                    <img
                        v-if="logoSrc"
                        :src="logoSrc"
                        :alt="t('projects.form_view.image_alt.logo')"
                        class="h-12 w-auto max-w-48 object-contain"
                    />
                    <input
                        :value="form.title"
                        type="text"
                        :aria-label="t('projects.form_view.placeholder.form_title')"
                        class="w-full rounded-md border-0 bg-transparent text-2xl font-bold text-gray-900 placeholder:text-gray-300 focus:ring-2 focus:ring-inset focus:ring-indigo-600"
                        :placeholder="t('projects.form_view.placeholder.form_title')"
                        @input="emit('change-settings', { title: $event.target.value })"
                    />
                    <textarea
                        :value="form.description"
                        rows="2"
                        :aria-label="t('projects.form_view.placeholder.form_description')"
                        class="w-full resize-none rounded-md border-0 bg-transparent text-sm text-gray-500 placeholder:text-gray-300 focus:ring-2 focus:ring-inset focus:ring-indigo-600"
                        :placeholder="t('projects.form_view.placeholder.form_description')"
                        @input="emit('change-settings', { description: $event.target.value })"
                    />
                </div>
            </div>

            <draggable
                :model-value="fields"
                item-key="header_name"
                handle=".form-field-drag-handle"
                class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm"
                @update:model-value="emit('reorder', $event)"
            >
                <template #item="{ element }">
                    <FormViewFieldRow
                        :field="element"
                        :field-name="fieldName(element)"
                        :fields="fields"
                        :workspace-id="form.workspace_id"
                        :table-id="form.table_id"
                        :link-tables="form.link_tables"
                        :incomplete="incompleteFields.includes(element.header_name)"
                        @update="emit('change-field', element.header_name, $event)"
                    />
                </template>
            </draggable>
        </div>

        <FormViewSettings
            :form="form"
            :cover-src="coverSrc"
            :logo-src="logoSrc"
            @update="emit('change-settings', $event)"
            @pick-image="(kind, file) => emit('pick-image', kind, file)"
            @remove-image="emit('remove-image', $event)"
        />
    </div>
</template>

<script setup>
import draggable from "vuedraggable";
import { t } from "@/i18n/index.js";
import { headerLabel } from "@/utils/projects/forms";
import FormCoverImage from "@/components/Projects/Form/FormCoverImage.vue";
import FormViewFieldRow from "@/components/Projects/Form/FormViewFieldRow.vue";
import FormViewSettings from "@/components/Projects/Form/FormViewSettings.vue";

const props = defineProps({
    form: {
        type: Object,
        required: true,
    },
    fields: {
        type: Array,
        required: true,
    },
    headers: {
        type: Array,
        default: () => [],
    },
    incompleteFields: {
        type: Array,
        default: () => [],
    },
    coverSrc: {
        type: String,
        default: "",
    },
    logoSrc: {
        type: String,
        default: "",
    },
});

const emit = defineEmits([
    "change-settings",
    "change-field",
    "reorder",
    "pick-image",
    "remove-image",
]);

function fieldName(field) {
    return headerLabel(
        props.headers.find((h) => h.name === field.header_name),
        t.value,
    );
}
</script>
