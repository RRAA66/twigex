<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div :class="halfWidth ? 'col-span-2 sm:col-span-1' : 'col-span-2'">
        <label :for="id" class="mb-1 block text-xs font-medium text-gray-600">
            {{ field.label || field.display_name || field.header_name }}
            <span v-if="field.required" class="text-red-500">*</span>
        </label>
        <p v-if="field.description" class="mb-1 text-xs text-gray-400">{{ field.description }}</p>

        <BaseCheckbox
            v-if="type === 'checkbox'"
            :model-value="Boolean(modelValue)"
            :label="field.placeholder || t('projects.form_view.placeholder.yes')"
            @update:model-value="update"
        />
        <DatePickerField
            v-else-if="type === 'date'"
            :id="id"
            :model-value="modelValue || ''"
            :placeholder="field.placeholder || t('projects.form_view.placeholder.pick_date')"
            @update:model-value="update"
        />
        <input
            v-else-if="type === 'color'"
            :id="id"
            type="color"
            :value="modelValue || '#000000'"
            class="h-9 w-16 cursor-pointer rounded-md border border-gray-300 bg-white p-1"
            @input="update($event.target.value)"
        />
        <FormLinkedPicker
            v-else-if="type === 'single-select' || type === 'link'"
            :id="id"
            :model-value="modelValue"
            :token="token"
            :table-id="linkedTableId(field)"
            :multiple="type === 'link'"
            :placeholder="
                field.placeholder ||
                t(
                    type === 'link'
                        ? 'projects.form_view.placeholder.select_options'
                        : 'projects.form_view.placeholder.select_option',
                )
            "
            @update:model-value="update"
        />
        <FormMemberPicker
            v-else-if="type === 'assignee'"
            :model-value="modelValue || ''"
            :members="members"
            :placeholder="field.placeholder || t('projects.form_view.placeholder.select_person')"
            @update:model-value="update"
        />
        <BaseSelect
            v-else-if="type === 'master-link'"
            :model-value="modelValue || ''"
            :options="tableOptions"
            :placeholder="field.placeholder || t('projects.form_view.placeholder.select_table')"
            @update:model-value="update"
        />
        <input
            v-else
            :id="id"
            :value="modelValue ?? ''"
            :type="inputType"
            :step="type === 'decimal' ? '0.01' : undefined"
            class="block w-full rounded-md border-0 px-3 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
            :class="error ? 'ring-red-600' : 'ring-gray-300'"
            :placeholder="field.placeholder || t(placeholderKey)"
            @input="update($event.target.value)"
        />

        <p v-if="error" class="mt-1 text-xs text-red-600">{{ error }}</p>
    </div>
</template>

<script setup>
import { computed, useId } from "vue";
import { t } from "@/i18n/index.js";
import { formFieldType, linkedTableId } from "@/utils/projects/forms";
import BaseCheckbox from "@/components/BaseCheckbox.vue";
import BaseSelect from "@/components/BaseSelect.vue";
import DatePickerField from "@/components/DatePicker/DatePickerField.vue";
import FormLinkedPicker from "@/components/Projects/Form/FormLinkedPicker.vue";
import FormMemberPicker from "@/components/Projects/Form/FormMemberPicker.vue";

const PLACEHOLDER_KEYS = {
    number: "projects.form_view.placeholder.number",
    decimal: "projects.form_view.placeholder.decimal",
    url: "projects.form_view.placeholder.url",
};

const props = defineProps({
    field: { type: Object, required: true },
    modelValue: { type: [String, Number, Boolean, Object, Array], default: null },
    token: { type: String, required: true },
    members: { type: Array, default: () => [] },
    linkTables: { type: Array, default: () => [] },
    error: { type: String, default: "" },
});

const emit = defineEmits(["update:modelValue"]);

const id = useId();

const type = computed(() => formFieldType(props.field));

const halfWidth = computed(() => ["left", "right"].includes(props.field.column_position));

const inputType = computed(() =>
    type.value === "decimal"
        ? "number"
        : ["number", "url"].includes(type.value)
          ? type.value
          : "text",
);

const placeholderKey = computed(
    () => PLACEHOLDER_KEYS[type.value] || "projects.form_view.placeholder.your_answer",
);

// The empty first option lets an answer be taken back.
const tableOptions = computed(() => [
    { value: "", label: t.value("projects.form_view.placeholder.select_table") },
    ...props.linkTables.map((table) => ({
        value: table.id,
        label: table.display_name || table.name,
    })),
]);

function update(value) {
    emit("update:modelValue", value);
}
</script>
