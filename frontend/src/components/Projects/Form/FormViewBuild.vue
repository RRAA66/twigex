<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="flex min-h-0 flex-1">
        <div class="flex-1 space-y-3 overflow-y-auto p-6">
            <div
                class="space-y-2 rounded-lg border border-t-4 border-gray-200 bg-white p-6 shadow-sm"
                :style="{ borderTopColor: form.accent_color }"
            >
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
                        @update="emit('change-field', element.header_name, $event)"
                    />
                </template>
            </draggable>
        </div>

        <FormViewSettings :form="form" @update="emit('change-settings', $event)" />
    </div>
</template>

<script setup>
import draggable from "vuedraggable";
import { t } from "@/i18n/index.js";
import { headerLabel } from "@/utils/projects/forms";
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
});

const emit = defineEmits(["change-settings", "change-field", "reorder"]);

function fieldName(field) {
    return headerLabel(
        props.headers.find((h) => h.name === field.header_name),
        t.value,
    );
}
</script>
