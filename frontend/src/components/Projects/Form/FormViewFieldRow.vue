<!--
Copyright (c) 2022 Twigex, SIA.
SPDX-License-Identifier: AGPL-3.0-only
-->
<template>
    <div class="border-b border-gray-100 last:border-b-0">
        <div
            class="flex cursor-pointer select-none items-center justify-between gap-x-3 px-4 py-2 hover:bg-gray-50"
            @click="expanded = !expanded"
        >
            <div class="flex min-w-0 flex-1 items-center gap-x-1.5">
                <Bars2Icon
                    class="form-field-drag-handle h-4 w-4 shrink-0 cursor-grab text-gray-300"
                    aria-hidden="true"
                    @click.stop
                />
                <ChevronRightIcon
                    :class="[
                        'h-3 w-3 shrink-0 text-gray-400 transition-transform',
                        expanded ? 'rotate-90' : '',
                    ]"
                    aria-hidden="true"
                />
                <span
                    :class="[
                        'truncate text-sm',
                        field.visible ? 'text-gray-700' : 'text-gray-300 line-through',
                    ]"
                >
                    {{ field.label || fieldName }}
                </span>
                <span v-if="field.required" class="shrink-0 text-xs text-red-400">*</span>
            </div>
            <ToggleSwitch
                v-if="!isNameField"
                v-model="field.visible"
                :sr-label="t('projects.form_view.field.visible')"
                @click.stop
            />
        </div>

        <div v-if="expanded" class="grid grid-cols-2 gap-3 border-t border-gray-200 bg-gray-50 p-4">
            <div class="col-span-2">
                <label :for="`${id}-label`" :class="labelClass">
                    {{ t("projects.form_view.field.label") }}
                </label>
                <input
                    :id="`${id}-label`"
                    v-model="field.label"
                    type="text"
                    :class="inputClass"
                    :placeholder="fieldName"
                />
            </div>
            <div class="col-span-2">
                <label :for="`${id}-placeholder`" :class="labelClass">
                    {{ t("projects.form_view.field.placeholder_label") }}
                </label>
                <input
                    :id="`${id}-placeholder`"
                    v-model="field.placeholder"
                    type="text"
                    :class="inputClass"
                    :placeholder="t('projects.form_view.placeholder.your_answer')"
                />
            </div>
            <BaseSelect
                v-model="field.column_position"
                :label="t('projects.form_view.field.column_width')"
                :options="columnOptions"
            />
            <div v-if="!isNameField" class="flex items-end pb-1.5">
                <BaseCheckbox v-model="field.required" :label="t('projects.form_view.required')" />
            </div>
            <FormViewFieldCondition
                v-if="!isNameField"
                class="col-span-2"
                :field="field"
                :fields="fields"
                :workspace-id="workspaceId"
                :table-id="tableId"
            />
        </div>
    </div>
</template>

<script setup>
import { computed, ref, useId } from "vue";
import { Bars2Icon, ChevronRightIcon } from "@heroicons/vue/24/outline";
import { t } from "@/i18n/index.js";
import BaseCheckbox from "@/components/BaseCheckbox.vue";
import BaseSelect from "@/components/BaseSelect.vue";
import ToggleSwitch from "@/components/ToggleSwitch.vue";
import FormViewFieldCondition from "@/components/Projects/Form/FormViewFieldCondition.vue";

const props = defineProps({
    field: {
        type: Object,
        required: true,
    },
    fieldName: {
        type: String,
        default: "",
    },
    fields: {
        type: Array,
        default: () => [],
    },
    workspaceId: {
        type: String,
        required: true,
    },
    tableId: {
        type: String,
        required: true,
    },
});

const id = useId();

const labelClass = "block text-sm font-medium leading-6 text-gray-900";
const inputClass =
    "mt-2 block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6";

const expanded = ref(false);

const isNameField = computed(() => props.field.header_name === "name");

const columnOptions = computed(() => [
    { value: "full", label: t.value("projects.form_view.field.col_full") },
    { value: "left", label: t.value("projects.form_view.field.col_left") },
    { value: "right", label: t.value("projects.form_view.field.col_right") },
]);
</script>
